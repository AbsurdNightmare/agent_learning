package tool

import (
	"context"
	"encoding/json"
	"fmt"
)

type getServerStatusTool struct{}

func NewGetServerStatusTool() *getServerStatusTool {
	return &getServerStatusTool{}
}

type getServerStatusToolParam struct {
	IP string `json:"ip"`
}

func (st *getServerStatusTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "get_server_status",
		Description: "获取服务器状态",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ip": map[string]string{"type": "string"},
			},
			"required": []string{"ip"},
		},
		RequireApproval: false,
	}
}

func (st *getServerStatusTool) Execute(ctx context.Context, argumentsJSON string) (string, error) {
	p := getServerStatusToolParam{}
	err := json.Unmarshal([]byte(argumentsJSON), &p)
	if err != nil {
		return "", err
	}

	result := fmt.Sprintf("服务器 %s 状态：正常，CPU 占用 15%%", p.IP)
	return result, nil
}
