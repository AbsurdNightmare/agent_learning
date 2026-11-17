package main

import (
	"context"
	"fmt"
	"learning/model"
	"learning/search"
	"learning/tool"
	"log"
	"net/http"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	llmClient    *model.Client
	searchClient *search.Client
	corpus       []search.Document
	agentTools   []tool.Tool
	mcpSession   *mcp.ClientSession
)

func initApp() {
	llmConfig, err := model.LoadConfig()
	if err != nil {
		log.Fatal("LLM配置失败:", err)
	}
	llmClient = model.NewClient(llmConfig)

	embedConfig, err := search.LoadConfig()
	if err != nil {
		log.Fatal("Embedding配置失败:", err)
	}
	searchClient = search.NewClient(embedConfig)

	corpus, err = search.LoadCorpus()
	if err != nil {
		log.Fatal("语料库加载失败:", err)
	}

	fmt.Println("[系统] 正在启动并连接 MCP Server...")
	cmd := exec.Command("D:\\agent\\mcp-file-server\\mcp-file-server.exe")
	transport := &mcp.CommandTransport{Command: cmd}

	ctx := context.Background()
	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "my-agent-client", Version: "1.0.0"}, nil)
	mcpSession, err = mcpClient.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatalf("[系统] 连接 MCP Server 失败: %v", err)
	}

	mcpToolsList, err := mcpSession.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		log.Fatalf("[系统] 获取 MCP 工具失败: %v", err)
	}

	agentTools = []tool.Tool{
		tool.NewGetServerStatusTool(),
		tool.NewWelcomeUserTool(),
		tool.NewSearchDocsTool(searchClient, corpus),
	}

	for _, mt := range mcpToolsList.Tools {
		fmt.Printf("[系统] 成功加载 MCP 工具: %s\n", mt.Name)
		agentTools = append(agentTools, &tool.MCPTool{
			Session: mcpSession,
			McpObj:  mt,
		})
	}
	fmt.Println("系统初始化完成，工具已加载")
}

func main() {
	initApp()

	http.HandleFunc("/api/chat", handleChat)
	http.HandleFunc("/api/resume", handleResume)
	http.HandleFunc("/api/history", handleHistory)

	fmt.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
