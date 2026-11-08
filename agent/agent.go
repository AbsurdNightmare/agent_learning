package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
	client       *model.Client
	messages     []model.Message
	tools        map[string]tool.Tool
	schemas      []any
	currentState AgentState
}

func NewAgent(client *model.Client, tools []tool.Tool) *Agent {
	a := &Agent{
		client:   client,
		messages: make([]model.Message, 0),
		tools:    make(map[string]tool.Tool),
		schemas:  tool.Schemas(tools),
	}

	for _, t := range tools {
		a.tools[t.Info().Name] = t
	}
	a.messages = append(a.messages, model.Message{Role: "system", Content: systemPrompt})
	return a
}

func (a *Agent) loop(ctx context.Context) error {
	for range maxSteps {
		thinkingTitlePrinted := false
		answeringTitlePrinted := false

		reply, err := a.client.Chat(ctx, a.messages, a.schemas, func(delta model.Delta) {
			if delta.ReasoningContent != "" {
				if !thinkingTitlePrinted {
					fmt.Print("[思考中]: ")
					thinkingTitlePrinted = true
				}
				fmt.Print(delta.ReasoningContent)
			}
			if delta.Content != "" {
				if !answeringTitlePrinted {
					fmt.Print("[回答]: ")
					answeringTitlePrinted = true
				}
				fmt.Print(delta.Content)
			}
		})
		fmt.Println()

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return errors.New("Agent 任务请求超时，强制终止！")
			}
			return err
		}

		a.messages = append(a.messages, reply)

		if len(reply.ToolCalls) == 0 {
			return nil
		}

		if ctx.Err() != nil {
			return errors.New("任务超时，终止执行工具！")
		}

		for _, call := range reply.ToolCalls {
			result, err := a.execute(ctx, call, a.messages)
			if err != nil {
				if errors.Is(err, ErrInterrupted) {
					return nil
				}

				fmt.Printf("工具报错: %s\n", err)
				result = fmt.Sprintf("工具执行失败: %v\n", err)
			}

			a.messages = append(a.messages, model.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: call.ID,
			})
		}
	}

	return errors.New("超出最大重试限制，强制终止")
}

func (a *Agent) Run(ctx context.Context, input string) error {
	a.messages = append(a.messages, model.Message{Role: "user", Content: input})
	return a.loop(ctx)
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

func (a *Agent) Resume(ctx context.Context, decision string) error {
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
	return a.loop(ctx)
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
