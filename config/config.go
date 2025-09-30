package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	LLMAPI           string `json:"llm_api"`
	APIKey           string `json:"api_key"`
	Model            string `json:"model"`
	DefaultShell     string `json:"default_shell"`
	SafeExecution    bool   `json:"safe_execution"`
	LocalLLMEndpoint string `json:"local_llm_endpoint"`
}

func Load(customPath string) (*Config, error) {
	path := customPath
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, ".config", "oneliner", "config.json")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := createDefault(path); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

func createDefault(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	defaultConfig := Config{
		LLMAPI:           "openai",
		APIKey:           "",
		Model:            "gpt-4.1-nano",
		DefaultShell:     "bash",
		SafeExecution:    true,
		LocalLLMEndpoint: "http://localhost:8000/v1/completions",
	}

	data, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
