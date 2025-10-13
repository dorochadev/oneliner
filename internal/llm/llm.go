package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

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
		return &LocalLLM{
			Endpoint:       cfg.LocalLLMEndpoint,
			Model:          cfg.Model,
			RequestTimeout: time.Duration(cfg.RequestTimeout) * time.Second,
			ClientTimeout:  time.Duration(cfg.ClientTimeout) * time.Second,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported LLM API: %s", cfg.LLMAPI)
	}
}

// ─── LOCAL LLM

type LocalLLM struct {
	Endpoint       string
	Model          string
	RequestTimeout time.Duration
	ClientTimeout  time.Duration
}

type localLLMRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type localLLMResponse struct {
	Choices []struct {
		Text string `json:"text"`
	} `json:"choices"`
}

func (l *LocalLLM) GenerateCommand(prompt string) (string, error) {
	if l.Endpoint == "" {
		return "", fmt.Errorf(
			"Local LLM endpoint not configured.\n\n" +
				"Quick setup:\n" +
				"  → Run: oneliner setup\n\n" +
				"Or manually configure:\n" +
				"  → oneliner config set llm_api local\n" +
				"  → oneliner config set local_llm_endpoint http://localhost:8000/v1/completions\n" +
				"  → oneliner config set model llama3",
		)
	}

	// Detect endpoint type
	isLMStudioChat := strings.Contains(l.Endpoint, "/v1/chat/completions")
	isLMStudioCompletions := !isLMStudioChat && strings.Contains(l.Endpoint, "/v1/completions")
	isOllamaChat := strings.Contains(l.Endpoint, "/api/chat")
	isOllamaGenerate := strings.Contains(l.Endpoint, "/api/generate")

	var jsonData []byte
	var err error

	// 🧱 Build correct request payload based on endpoint
	if isOllamaGenerate {
		// Ollama /api/generate endpoint
		jsonData, err = json.Marshal(map[string]any{
			"model":  l.Model,
			"prompt": prompt,
			"stream": false,
		})
	} else if isOllamaChat {
		// Ollama /api/chat endpoint
		jsonData, err = json.Marshal(map[string]any{
			"model": l.Model,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
			"stream": false,
		})
	} else if isLMStudioChat {
		// LM Studio /v1/chat/completions endpoint (OpenAI-compatible)
		jsonData, err = json.Marshal(map[string]any{
			"model": l.Model,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
			"max_tokens":  512,
			"temperature": 0.7,
			"stream":      false,
		})
	} else if isLMStudioCompletions {
		// LM Studio /v1/completions endpoint
		jsonData, err = json.Marshal(map[string]any{
			"model":       l.Model,
			"prompt":      prompt,
			"max_tokens":  512,
			"temperature": 0.7,
			"stream":      false,
		})
	} else {
		// Default: try OpenAI-compatible chat format (most common)
		jsonData, err = json.Marshal(map[string]any{
			"model": l.Model,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
			"max_tokens":  512,
			"temperature": 0.7,
			"stream":      false,
		})
	}

	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	// Timeouts
	timeout := l.RequestTimeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	clientTimeout := l.ClientTimeout
	if clientTimeout == 0 {
		clientTimeout = 65 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", l.Endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: clientTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Try to detect if response is streaming NDJSON (Ollama)
	// we will look at the content to see if it looks like NDJSON
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	// Check if it's NDJSON by looking for newline-separated JSON objects
	if isOllamaGenerate || isOllamaChat {
		lines := strings.Split(string(bodyBytes), "\n")
		var textBuilder strings.Builder
		foundResponse := false

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			var msg map[string]any
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				continue // Not valid JSON, skip
			}

			foundResponse = true

			// Ollama /api/generate response format
			if part, ok := msg["response"].(string); ok {
				textBuilder.WriteString(part)
			}

			// Ollama /api/chat response format
			if message, ok := msg["message"].(map[string]any); ok {
				if content, ok := message["content"].(string); ok {
					textBuilder.WriteString(content)
				}
			}

			// Check if this is the final message
			if done, ok := msg["done"].(bool); ok && done {
				break
			}
		}

		if foundResponse {
			out := strings.TrimSpace(textBuilder.String())
			if out == "" {
				return "", fmt.Errorf("empty response from local LLM")
			}
			return out, nil
		}
	}

	// Parse as standard JSON response

	// LM Studio / OpenAI completions format
	var lmCompletion struct {
		Choices []struct {
			Text string `json:"text"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(bodyBytes, &lmCompletion); err == nil && len(lmCompletion.Choices) > 0 {
		if txt := strings.TrimSpace(lmCompletion.Choices[0].Text); txt != "" {
			return txt, nil
		}
	}

	// OpenAI-style chat completions format (LM Studio /v1/chat/completions)
	var openAIChat struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(bodyBytes, &openAIChat); err == nil && len(openAIChat.Choices) > 0 {
		if msg := strings.TrimSpace(openAIChat.Choices[0].Message.Content); msg != "" {
			return msg, nil
		}
	}

	// Ollama non-streaming response
	var ollama struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Response string `json:"response"`
	}
	if err := json.Unmarshal(bodyBytes, &ollama); err == nil {
		if ollama.Message.Content != "" {
			return strings.TrimSpace(ollama.Message.Content), nil
		}
		if ollama.Response != "" {
			return strings.TrimSpace(ollama.Response), nil
		}
	}

	return "", fmt.Errorf("empty response from local LLM: %s", string(bodyBytes))
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
		return "", fmt.Errorf(
			"OpenAI API key not configured.\n\n" +
				"Quick setup:\n" +
				"  → Run: oneliner setup\n\n" +
				"Or manually configure:\n" +
				"  → oneliner config set llm_api openai\n" +
				"  → oneliner config set api_key sk-xxxx\n" +
				"  → oneliner config set model gpt-4o\n\n" +
				"Get your API key: https://platform.openai.com/api-keys",
		)
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
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

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
		return "", fmt.Errorf(
			"Claude API key not configured.\n\n" +
				"Quick setup:\n" +
				"  → Run: oneliner setup\n\n" +
				"Or manually configure:\n" +
				"  → oneliner config set llm_api claude\n" +
				"  → oneliner config set api_key sk-ant-xxxx\n" +
				"  → oneliner config set model claude-sonnet-4-5-20250929\n\n" +
				"Get your API key: https://console.anthropic.com/",
		)
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
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

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
