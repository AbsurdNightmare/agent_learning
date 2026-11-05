package tool

import "context"

type ToolInfo struct {
	Name            string
	Description     string
	Parameters      map[string]any
	RequireApproval bool
}

type Tool interface {
	Info() ToolInfo
	Execute(ctx context.Context, argumentsJSON string) (string, error)
}

func Schema(t Tool) map[string]any {
	info := t.Info()
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":             info.Name,
			"description":      info.Description,
			"parameters":       info.Parameters,
			"require_approval": info.RequireApproval,
		},
	}
}

func Schemas(tools []Tool) []any {
	schemas := make([]any, 0, len(tools))
	for _, t := range tools {
		schemas = append(schemas, Schema(t))
	}
	return schemas
}
