package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type deleteFileTool struct{}

func NewDeleteFileTool() *deleteFileTool {
	return &deleteFileTool{}
}

type deleteFileToolParam struct {
	Path string `json:"path"`
}

func (dt *deleteFileTool) Info() ToolInfo {
	return ToolInfo{
		Name:        "delete_file",
		Description: "删除目标文件",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]string{"type": "string"},
			},
			"required": []string{"path"},
		},
		RequireApproval: true,
	}
}

func (dt *deleteFileTool) Execute(ctx context.Context, argumentsJSON string) (string, error) {
	p := deleteFileToolParam{}
	err := json.Unmarshal([]byte(argumentsJSON), &p)
	if err != nil {
		return "", err
	}

	err = os.Remove(p.Path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("文件 %s 已成功删除", p.Path), nil
}
