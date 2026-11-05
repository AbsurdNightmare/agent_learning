package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"learning/agent"
	"learning/model"
	"learning/tool"
)

func main() {
	config, err := model.LoadConfig()
	if err != nil {
		fmt.Println("加载模型配置失败:", err)
		return
	}

	a := agent.NewAgent(model.NewClient(config), []tool.Tool{
		tool.NewGetServerStatusTool(),
		tool.NewWelcomeUserTool(),
		tool.NewDeleteFileTool(),
	})

	scanner := bufio.NewScanner(os.Stdin)
	for {
		if _, err := os.Stat("checkpoint.json"); err == nil {
			fmt.Print("[等待审批 (y/n)] >>  ")
		} else {
			fmt.Print(">>  ")
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

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		var runErr error
		if _, err := os.Stat("checkpoint.json"); err == nil {
			runErr = a.Resume(ctx, input)
		} else {
			runErr = a.Run(ctx, input)
		}

		cancel()

		if runErr != nil {
			fmt.Println(runErr)
		}
	}
}
