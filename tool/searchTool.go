package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"learning/search"
	"strings"
)

const topK = 3
const maxChars = 3000

type searchDocsTool struct {
	client *search.Client
	corpus []search.Document
}

func NewSearchDocsTool(client *search.Client, corpus []search.Document) *searchDocsTool {
	return &searchDocsTool{
		client: client,
		corpus: corpus,
	}
}

type searchDocsToolParam struct {
	Query string `json:"query"`
}

func (st *searchDocsTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "search_docs",
		Description: "当需要查找技术文档、代码说明或内部资料时使用此工具。输入查询关键词或问题，返回最相关的文档片段。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "要检索的关键词或自然语言问题",
				},
			},
			"required": []string{"query"},
		},
		RequireApproval: false,
	}
}

func (st *searchDocsTool) Execute(ctx context.Context, argumentsJSON string) (string, error) {
	p := searchDocsToolParam{}
	err := json.Unmarshal([]byte(argumentsJSON), &p)
	if err != nil {
		return "", err
	}

	results, err := st.client.HybridSearch(ctx, p.Query, st.corpus)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("为你找到以下相关参考文档：\n\n")

	limit := min(len(results), topK)

	for i := range limit {
		sb.WriteString(fmt.Sprintf("【文档来源：%s】\n", results[i].Doc.Name))
		sb.WriteString(truncateContent(results[i].Doc.Content))
		sb.WriteString("\n\n")
	}

	return sb.String(), nil
}

func truncateContent(content string) string {
	if len(content) <= maxChars {
		return content
	}

	paragraphs := strings.Split(content, "\n\n")

	var result string
	for _, para := range paragraphs {
		if len(result)+len(para) > maxChars {
			result += "..."
			break
		}
		result += para
	}
	return result
}
