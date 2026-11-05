package model

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

const retryInterval = 3 * time.Second

type Client struct {
	config Config
	http   *http.Client
}

func NewClient(config Config) *Client {
	return &Client{
		config: config,
		http: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

func (c *Client) Chat(ctx context.Context, messages []Message, tools []any, onDelta func(Delta)) (Message, error) {
	res, err := c.send(ctx, messages, tools)
	if err != nil {
		return Message{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		content, _ := io.ReadAll(res.Body)
		return Message{}, fmt.Errorf("API报错，状态码: %d, 内容: %s", res.StatusCode, content)
	}

	return readStream(res.Body, onDelta)
}

func (c *Client) send(ctx context.Context, messages []Message, tools []any) (*http.Response, error) {
	payload, err := json.Marshal(Request{
		Model:    c.config.Model,
		Messages: messages,
		Tools:    tools,
		Stream:   true,
	})
	if err != nil {
		return nil, err
	}

	for range c.config.MaxRetries {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

		res, err := c.http.Do(req)
		if err == nil {
			return res, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		fmt.Printf("网络错误: %v，正在重试\n", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(retryInterval):
		}
	}

	return nil, errors.New("网络请求失败，超出最大重试次数")
}

func readStream(body io.Reader, onDelta func(Delta)) (Message, error) {
	scanner := bufio.NewScanner(body)
	toolCallMap := make(map[int]*ToolCall)
	var fullContent string
	var fullReasoningContent string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		dataStr := strings.TrimPrefix(line, "data: ")
		if dataStr == "[DONE]" {
			break
		}

		var chunk StreamChunk
		if err := json.Unmarshal([]byte(dataStr), &chunk); err != nil {
			fmt.Println("解析单行失败:", err)
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta
		if onDelta != nil {
			onDelta(delta)
		}

		fullContent += delta.Content
		fullReasoningContent += delta.ReasoningContent

		for _, t := range delta.ToolCalls {
			call, exists := toolCallMap[t.Index]
			if !exists {
				call = &ToolCall{Type: "function"}
				toolCallMap[t.Index] = call
			}
			if t.ID != "" {
				call.ID = t.ID
			}
			if t.Type != "" {
				call.Type = t.Type
			}
			if t.Function.Name != "" {
				call.Function.Name = t.Function.Name
			}
			call.Function.Arguments += t.Function.Arguments
		}
	}

	if err := scanner.Err(); err != nil {
		return Message{}, err
	}

	indexes := make([]int, 0, len(toolCallMap))
	for index := range toolCallMap {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	finalToolCalls := make([]ToolCall, 0, len(indexes))
	for _, index := range indexes {
		finalToolCalls = append(finalToolCalls, *toolCallMap[index])
	}

	return Message{
		Role:             "assistant",
		Content:          fullContent,
		ReasoningContent: fullReasoningContent,
		ToolCalls:        finalToolCalls,
	}, nil
}
