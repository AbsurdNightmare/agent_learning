package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type MCPTool struct {
	Session *mcp.ClientSession
	McpObj  *mcp.Tool
}

func (m *MCPTool) Info() ToolInfo {
	schemaBytes, _ := json.Marshal(m.McpObj.InputSchema)
	var params map[string]any
	_ = json.Unmarshal(schemaBytes, &params)

	needsApproval := strings.Contains(m.McpObj.Name, "delete")

	return ToolInfo{
		Name:            m.McpObj.Name,
		Description:     m.McpObj.Description,
		Parameters:      params,
		RequireApproval: needsApproval,
	}
}

func (m *MCPTool) Execute(ctx context.Context, argumentsJSON string) (string, error) {
	var args map[string]any
	if err := json.Unmarshal([]byte(argumentsJSON), &args); err != nil {
		return "", err
	}

	res, err := m.Session.CallTool(ctx, &mcp.CallToolParams{
		Name:      m.McpObj.Name,
		Arguments: args,
	})
	if err != nil {
		return "", err
	}

	var textResult string
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			textResult += tc.Text
		}
	}

	if textResult == "" {
		return fmt.Sprintf("工具 %s 执行完毕，但没有返回文本结果", m.McpObj.Name), nil
	}
	return textResult, nil
}
