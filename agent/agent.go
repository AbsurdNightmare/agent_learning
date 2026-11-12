package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"learning/model"
	"learning/tool"
)

const maxSteps = 10

const logPath = "execution_log.txt"

var ErrInterrupted = errors.New("agent interrupted for approval")

type AgentState string

const (
	StateRunning         AgentState = "RUNNING"
	StatePendingApproval AgentState = "PENDING_APPROVAL"
	StateApproved        AgentState = "APPROVED"
	StateResumed         AgentState = "RESUMED"
	StateRejected        AgentState = "REJECTED"
	StateTerminated      AgentState = "TERMINATED"
)

type Agent struct {
	client           *model.Client
	messages         []model.Message
	tools            map[string]tool.Tool
	schemas          []any
	currentState     AgentState
	maxContextTokens int
}

func NewAgent(client *model.Client, tools []tool.Tool) *Agent {
	maxTokens, _ := strconv.Atoi(os.Getenv("MAX_CONTEXT_TOKENS"))

	a := &Agent{
		client:           client,
		messages:         make([]model.Message, 0),
		tools:            make(map[string]tool.Tool),
		schemas:          tool.Schemas(tools),
		maxContextTokens: maxTokens,
	}

	for _, t := range tools {
		a.tools[t.Info().Name] = t
	}
	a.messages = append(a.messages, model.Message{Role: "system", Content: systemPrompt})
	return a
}

func (a *Agent) loop(ctx context.Context, eventChan chan<- model.AgentEvent) error {
	for range maxSteps {
		a.messages = a.CompressHistory(a.messages)

		eventChan <- model.AgentEvent{Type: "thinking", Content: "开始请求模型..."}

		reply, err := a.client.Chat(ctx, a.messages, a.schemas, func(delta model.Delta) {
			if delta.ReasoningContent != "" {
				eventChan <- model.AgentEvent{Type: "thinking", Content: delta.ReasoningContent}
			}
			if delta.Content != "" {
				eventChan <- model.AgentEvent{Type: "thinking", Content: delta.Content}
			}
		})

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return errors.New("Agent 任务请求超时，强制终止！")
			}
			return err
		}

		a.messages = append(a.messages, reply)

		if len(reply.ToolCalls) == 0 {
			eventChan <- model.AgentEvent{Type: "done", Content: "本轮任务结束"}
			return nil
		}

		if ctx.Err() != nil {
			return errors.New("任务超时，终止执行工具！")
		}

		for _, call := range reply.ToolCalls {
			eventChan <- model.AgentEvent{Type: "tool_call", Content: "调用工具: " + call.Function.Name}

			result, err := a.execute(ctx, call, a.messages)
			if err != nil {
				if errors.Is(err, ErrInterrupted) {
					eventChan <- model.AgentEvent{Type: "interrupted", Content: "触发审批，等待恢复"}
					return nil
				}

				result = fmt.Sprintf("工具执行失败: %v\n", err)
			}

			eventChan <- model.AgentEvent{Type: "tool_result", Content: result}
			a.messages = append(a.messages, model.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: call.ID,
			})
		}
	}

	return errors.New("超出最大重试限制，强制终止")
}

func (a *Agent) Run(ctx context.Context, input string, eventChan chan<- model.AgentEvent) error {
	a.messages = append(a.messages, model.Message{Role: "user", Content: input})
	return a.loop(ctx, eventChan)
}

func (a *Agent) execute(ctx context.Context, call model.ToolCall, messages []model.Message) (string, error) {
	t, exists := a.tools[call.Function.Name]
	if !exists {
		return "", fmt.Errorf("不存在的工具: %s", call.Function.Name)
	}

	if t.Info().RequireApproval {
		fmt.Println("[需要审批]: 检测到高危操作，挂起等待审批！")

		cp := &CheckPoint{
			RunID:           fmt.Sprintf("run-%d", time.Now().UnixNano()),
			StepID:          int(time.Now().UnixNano()),
			CurrentState:    StatePendingApproval,
			Messages:        messages,
			PendingToolCall: call,
		}
		err := SaveCheckPoint(ctx, cp)
		if err != nil {
			return "", err
		}

		a.currentState = StatePendingApproval
		return "", ErrInterrupted
	}

	fmt.Printf("[执行工具] %s，参数: %s\n", call.Function.Name, call.Function.Arguments)
	return t.Execute(ctx, call.Function.Arguments)
}

func (a *Agent) Resume(ctx context.Context, decision string, eventChan chan<- model.AgentEvent) error {
	cp, err := LoadCheckPoint(ctx)
	if err != nil {
		return fmt.Errorf("没有找到需要恢复的任务: %v", err)
	}

	if cp.CurrentState != StatePendingApproval {
		return fmt.Errorf("当前没有等待审核的任务")
	}

	a.messages = cp.Messages
	call := cp.PendingToolCall

	var result string
	if decision == "n" {
		fmt.Println("[用户选择]: 用户拒绝了操作")
		a.currentState = StateRejected
		result = "用户拒绝了此危险操作，请勿重试"
		a.currentState = StateTerminated
	} else if decision == "y" {
		fmt.Println("[用户选择]: 用户批准了操作")
		a.currentState = StateApproved

		if isAlreadyExecuted(cp.RunID, cp.StepID) {
			fmt.Println("检测到该步骤已执行过，触发幂等保护，直接跳过！")
			result = "该工具已在此前成功执行"
		} else {
			t := a.tools[call.Function.Name]
			result, err = t.Execute(ctx, call.Function.Arguments)
			if err != nil {
				result = fmt.Sprintf("工具执行失败: %v", err)
			}

			recordExecution(cp.RunID, cp.StepID)
		}

		a.currentState = StateResumed
	} else if newArgs, ok := strings.CutPrefix(decision, "modify "); ok {
		if !json.Valid([]byte(newArgs)) {
			return fmt.Errorf("您修改的参数不是合法的 JSON 格式")
		}
		fmt.Println("[用户选择]: 修改参数为:", newArgs)
		a.currentState = StateApproved

		call.Function.Arguments = newArgs

		t := a.tools[call.Function.Name]
		result, err = t.Execute(ctx, call.Function.Arguments)
		if err != nil {
			result = fmt.Sprintf("工具执行失败: %v", err)
		}
		result = fmt.Sprintf("用户在审批时已将当前参数修改成 %s\n %s", newArgs, result)

		a.currentState = StateResumed
	} else {
		return fmt.Errorf("未知的指令，请输入 y 或 n")
	}

	a.messages = append(a.messages, model.Message{
		Role:       "tool",
		Content:    result,
		ToolCallID: call.ID,
	})

	_ = os.Remove("checkpoint.json")

	a.currentState = StateRunning
	return a.loop(ctx, eventChan)
}

func (a *Agent) CompressHistory(messages []model.Message) []model.Message {
	historyLen := 0
	for _, m := range messages {
		historyLen += (len(m.Content) + len(m.ReasoningContent)) / 3
	}

	if historyLen < a.maxContextTokens {
		return messages
	}

	fmt.Println("[上下文]: 上下文超出窗口大小，自动开启上下文压缩...")

	var compressed []model.Message
	compressed = append(compressed, messages[0])
	if len(messages) > 1 && messages[1].Role == "user" {
		compressed = append(compressed, messages[1])
	}

	compressed = append(compressed, model.Message{
		Role:    "user",
		Content: "为节省上下文，中间的执行历史已被压缩裁剪，请基于最新的工具结果和用户初始任务作答",
	})

	startIndex := max(len(messages)-6, 2)

	for startIndex < len(messages) && messages[startIndex].Role == "tool" {
		startIndex--
	}

	compressed = append(compressed, messages[startIndex:]...)
	fmt.Println("[上下文]: 上下文压缩完毕，继续执行")
	return compressed
}

func isAlreadyExecuted(runID string, stepID int) bool {
	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		fmt.Println("读取账本失败:", err)
		return false
	}
	defer file.Close()

	targetKey := fmt.Sprintf("%s-%d", runID, stepID)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if scanner.Text() == targetKey {
			return true
		}
	}
	return false
}

func recordExecution(runID string, stepID int) {
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("无法打开日志账本:", err)
		return
	}
	defer file.Close()

	log := fmt.Sprintf("%s-%d\n", runID, stepID)
	if _, err := file.WriteString(log); err != nil {
		fmt.Println("写入账本失败:", err)
	}
}
