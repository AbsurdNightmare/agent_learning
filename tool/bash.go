package tool

import (
	"context"
	"encoding/json"
	"os/exec"
	"runtime"
)

type bashTool struct{}

func NewBashTool() *bashTool {
	return &bashTool{}
}

type bashToolParam struct {
	Command string `json:"command"`
}

func (bt *bashTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "bash",
		Description: "执行终端命令",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]string{"type": "string"},
			},
			"required": []string{"command"},
		},
		RequireApproval: false,
	}
}

func (bt *bashTool) Execute(ctx context.Context, argumentsJSON string) (string, error) {
	p := bashToolParam{}
	err := json.Unmarshal([]byte(argumentsJSON), &p)
	if err != nil {
		return "", err
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", p.Command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", p.Command)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
