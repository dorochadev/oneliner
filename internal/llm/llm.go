package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/dorochadev/oneliner/config"
)

type LLM interface {
	GenerateCommand(prompt string) (string, error)
}

func New(cfg *config.Config) (LLM, error) {
	switch cfg.LLMAPI {
	case "openai":
		return &OpenAI{
			APIKey: cfg.APIKey,
			Model:  cfg.Model,
		}, nil
	case "claude":
		return &Claude{
			APIKey:    cfg.APIKey,
			Model:     cfg.Model,
			MaxTokens: cfg.ClaudeMaxTokens,
		}, nil
	case "local":
		if cfg.LocalLLMEndpoint == "" {
			return nil, fmt.Errorf("local_llm_endpoint must be set in config for local LLM usage")
		}
		return &LocalLLM{
			Endpoint: cfg.LocalLLMEndpoint,
			Model:    cfg.Model,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported LLM API: %s", cfg.LLMAPI)
	}
}

// ─── LOCAL LLM

type LocalLLM struct {
	Endpoint string
	Model    string
}

type localLLMRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type localLLMResponse struct {
	Result string `json:"result"`
}

func (l *LocalLLM) GenerateCommand(prompt string) (string, error) {
	if l.Endpoint == "" {
		return "", fmt.Errorf("local LLM endpoint not configured")
	}

	reqBody := localLLMRequest{
		Model:  l.Model,
		Prompt: prompt,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", l.Endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("local LLM error: %s", string(body))
	}

	var result localLLMResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result.Result == "" {
		return "", fmt.Errorf("no response from local LLM")
	}

	return result.Result, nil
}

// ─── OPENAI

type OpenAI struct {
	APIKey string
	Model  string
}

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
}

func (o *OpenAI) GenerateCommand(prompt string) (string, error) {
	if o.APIKey == "" {
		return "", fmt.Errorf("OpenAI API key not configured")
	}

	reqBody := openAIRequest{
		Model: o.Model,
		Messages: []openAIMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API error: %s", string(body))
	}

	var result openAIResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return result.Choices[0].Message.Content, nil
}

// ─── CLAUDE

type Claude struct {
	APIKey    string
	Model     string
	MaxTokens int
}

type claudeRequest struct {
	Model     string          `json:"model"`
	Messages  []claudeMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func (c *Claude) GenerateCommand(prompt string) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("Claude API key not configured")
	}

	maxTokens := c.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024 // fallback default
	}

	reqBody := claudeRequest{
		Model: c.Model,
		Messages: []claudeMessage{
			{Role: "user", Content: prompt},
		},
		MaxTokens: maxTokens,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Claude API error: %s", string(body))
	}

	var result claudeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("no response from Claude")
	}

	return result.Content[0].Text, nil
}
