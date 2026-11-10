package model

import (
	"errors"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	BaseURL    string
	APIKey     string
	Model      string
	Timeout    time.Duration
	MaxRetries int
	MCPPath    string
}

func NewConfig(baseURL, apiKey, modelName, mcpPath string) Config {
	return Config{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		Model:      modelName,
		Timeout:    15 * time.Second,
		MaxRetries: 10,
		MCPPath:    mcpPath,
	}
}

func LoadConfig() (Config, error) {
	_ = godotenv.Load()
	config := NewConfig(os.Getenv("LLM_URL"), os.Getenv("LLM_API_KEY"), os.Getenv("LLM_MODEL"), os.Getenv("MCP_PATH"))
	return config, config.Validate()
}

func (c Config) Validate() error {
	if c.BaseURL == "" {
		return errors.New("缺少 LLM_URL 配置")
	}
	if c.APIKey == "" {
		return errors.New("缺少 LLM_API_KEY 配置")
	}
	if c.Model == "" {
		return errors.New("缺少 LLM_MODEL 配置")
	}
	if c.MCPPath == "" {
		return errors.New("缺少 MCP_PATH 配置")
	}
	return nil
}
