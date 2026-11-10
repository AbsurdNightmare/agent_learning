package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"learning/agent"
	"learning/model"
	"learning/tool"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	config, err := model.LoadConfig()
	if err != nil {
		fmt.Println("加载模型配置失败:", err)
		return
	}

	ctx := context.Background()

	fmt.Println("[系统] 正在启动并连接 MCP Server...")
	cmd := exec.Command(config.MCPPath)
	transport := &mcp.CommandTransport{Command: cmd}

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "my-agent-client", Version: "1.0.0"}, nil)
	session, err := mcpClient.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatalf("[系统] 连接 MCP Server 失败: %v", err)
	}
	defer session.Close()

	mcpToolsList, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		log.Fatalf("[系统] 获取 MCP 工具失败: %v", err)
	}

	agentTools := []tool.Tool{
		tool.NewGetServerStatusTool(),
		tool.NewWelcomeUserTool(),
	}

	for _, mt := range mcpToolsList.Tools {
		fmt.Printf("[系统] 成功加载 MCP 工具: %s\n", mt.Name)
		agentTools = append(agentTools, &tool.MCPTool{
			Session: session,
			McpObj:  mt,
		})
	}

	a := agent.NewAgent(model.NewClient(config), agentTools)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		cp, err := agent.LoadCheckPoint(ctx)
		isPending := err == nil && cp.CurrentState == agent.StatePendingApproval
		if isPending {
			fmt.Print("[等待审批 (y/n/modify)] >> ")
		} else {
			fmt.Print(">> ")
		}

		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "exit" {
			break
		}

		taskCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		var runErr error
		if isPending {
			runErr = a.Resume(taskCtx, input)
		} else {
			runErr = a.Run(taskCtx, input)
		}

		cancel()

		if runErr != nil {
			fmt.Println(runErr)
		}
	}
}
