package config

import (
	"encoding/json"
	"errors"
	"os"
)

type Config struct {
	Port   string       `json:"port"`
	OpenAI OpenAIConfig `json:"openai"`
}

type OpenAIConfig struct {
	APIKey     string `json:"api_key"`
	BaseURL    string `json:"base_url"`
	ChatModel  string `json:"chat_model"`
	TimeoutSec int    `json:"timeout_sec"`
}

func Load(path string) (Config, error) {
	cfg := defaultConfig()

	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, err
	}

	if len(b) == 0 {
		return cfg, nil
	}

	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, err
	}

	if cfg.OpenAI.BaseURL == "" {
		cfg.OpenAI.BaseURL = "https://api.openai.com/v1"
	}
	if cfg.OpenAI.ChatModel == "" {
		cfg.OpenAI.ChatModel = "gpt-4o-mini"
	}
	if cfg.OpenAI.TimeoutSec <= 0 {
		cfg.OpenAI.TimeoutSec = 60
	}

	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		Port: "8080",
		OpenAI: OpenAIConfig{
			BaseURL:    "https://api.openai.com/v1",
			ChatModel:  "gpt-4o-mini",
			TimeoutSec: 60,
		},
	}
}
