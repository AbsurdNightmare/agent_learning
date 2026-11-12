package main

import (
	"fmt"
	"learning/model"
	"learning/search"
	"learning/tool"
	"log"
	"net/http"
)

var (
	llmClient    *model.Client
	searchClient *search.Client
	corpus       []search.Document
	agentTools   []tool.Tool
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

	agentTools = []tool.Tool{
		tool.NewSearchDocsTool(searchClient, corpus),
		tool.NewWelcomeUserTool(),
		tool.NewGetServerStatusTool(),
	}
	fmt.Println("系统初始化完成，工具已加载")
}

func main() {
	initApp()

	http.HandleFunc("/api/chat", handleChat)
	http.HandleFunc("/api/resume", handleResume)

	fmt.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
