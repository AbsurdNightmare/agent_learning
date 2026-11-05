package tool

import (
	"context"
	"encoding/json"
	"fmt"
)

type welcomeUserTool struct{}

func NewWelcomeUserTool() *welcomeUserTool {
	return &welcomeUserTool{}
}

type welcomeUserToolParam struct {
	Name string `json:"name"`
}

func (wt *welcomeUserTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "welcome_user",
		Description: "向用户问好",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]string{"type": "string"},
			},
			"required": []string{"name"},
		},
		RequireApproval: false,
	}
}

func (wt *welcomeUserTool) Execute(ctx context.Context, argumentsJSON string) (string, error) {
	p := welcomeUserToolParam{}
	err := json.Unmarshal([]byte(argumentsJSON), &p)
	if err != nil {
		return "", err
	}

	result := fmt.Sprintf("热烈欢迎 %s 使用本Agent!\n", p.Name)
	return result, nil
}
