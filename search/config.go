package search

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	URL    string
	APIKey string
	Model  string
}

func NewConfig(baseURL, apiKey, modelName string) Config {
	return Config{
		URL:    baseURL,
		APIKey: apiKey,
		Model:  modelName,
	}
}

func LoadConfig() (Config, error) {
	_ = godotenv.Load()
	config := NewConfig(os.Getenv("EMBEDDING_URL"), os.Getenv("EMBEDDING_API_KEY"), os.Getenv("EMBEDDING_MODEL"))
	return config, config.Validate()
}

func (c Config) Validate() error {
	if c.URL == "" {
		return errors.New("缺少 EMBEDDING_URL 配置")
	}
	if c.APIKey == "" {
		return errors.New("缺少 EMBEDDING_API_KEY 配置")
	}
	if c.Model == "" {
		return errors.New("缺少 EMBEDDING_MODEL 配置")
	}
	return nil
}
