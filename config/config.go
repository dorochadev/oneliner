package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Config struct {
	LLMAPI              string   `json:"llm_api"`
	APIKey              string   `json:"api_key"`
	Model               string   `json:"model"`
	DefaultShell        string   `json:"default_shell"`
	LocalLLMEndpoint    string   `json:"local_llm_endpoint"`
	ClaudeMaxTokens     int      `json:"claude_max_tokens"`
	RequestTimeout      int      `json:"request_timeout"`
	ClientTimeout       int      `json:"client_timeout"`
	BlacklistedBinaries []string `json:"blacklisted_binaries"`
}

// Load loads config from disk, ensuring any missing fields are added.
func Load(customPath string) (*Config, error) {
	path := resolvePath(customPath)

	// If no config exists, create a fresh one.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := createDefault(path); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Decode into map first to detect missing keys.
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Decode again into typed struct.
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file into struct: %w", err)
	}

	def := defaultConfig()
	updated := false

	// Check and patch missing or zero-value fields.
	// --- Strings ---
	if strings.TrimSpace(cfg.LLMAPI) == "" {
		cfg.LLMAPI = def.LLMAPI
		updated = true
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		cfg.APIKey = def.APIKey
		updated = true
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = def.Model
		updated = true
	}
	if strings.TrimSpace(cfg.DefaultShell) == "" {
		cfg.DefaultShell = def.DefaultShell
		updated = true
	}
	if strings.TrimSpace(cfg.LocalLLMEndpoint) == "" {
		cfg.LocalLLMEndpoint = def.LocalLLMEndpoint
		updated = true
	}

	// --- Integers ---
	if cfg.ClaudeMaxTokens == 0 {
		cfg.ClaudeMaxTokens = def.ClaudeMaxTokens
		updated = true
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = def.RequestTimeout
		updated = true
	}
	if cfg.ClientTimeout == 0 {
		cfg.ClientTimeout = def.ClientTimeout
		updated = true
	}

	// --- Slice ---
	if len(cfg.BlacklistedBinaries) == 0 {
		cfg.BlacklistedBinaries = def.BlacklistedBinaries
		updated = true
	}

	// --- Automatic new-field detection ---
	defMap := structToMap(def)
	for k := range defMap {
		if _, ok := raw[k]; !ok {
			updated = true
			break
		}
	}

	// Save back if updated or new fields detected.
	if updated {
		if err := Save(path, &cfg); err != nil {
			return nil, fmt.Errorf("failed to update config: %w", err)
		}
	}

	return &cfg, nil
}

func Save(path string, cfg *Config) error {
	path = resolvePath(path)

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

func createDefault(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	def := defaultConfig()

	data, err := json.MarshalIndent(def, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func defaultConfig() Config {
	return Config{
		LLMAPI:           "openai",
		APIKey:           "",
		Model:            "gpt-4.1-nano",
		DefaultShell:     detectDefaultShell(),
		LocalLLMEndpoint: "http://localhost:8000/v1/completions",
		ClaudeMaxTokens:  1024,
		RequestTimeout:   60,
		ClientTimeout:    65,
		BlacklistedBinaries: []string{
			"rm", "dd", "mkfs", "fdisk", "parted",
			"shred", "curl", "wget", "nc", "ncat",
		},
	}
}

func detectDefaultShell() string {
	goos := strings.ToLower(runtime.GOOS)
	switch goos {
	case "windows":
		if os.Getenv("WSL_DISTRO_NAME") != "" {
			return "bash"
		}
		return "powershell"
	case "darwin", "linux":
		return "bash"
	default:
		return "bash"
	}
}

func resolvePath(customPath string) string {
	if customPath != "" {
		return customPath
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// fallback to current directory if home can’t be resolved
		return "./config.json"
	}
	return filepath.Join(home, ".config", "oneliner", "config.json")
}

// --- helper to convert struct -> map[string]any for auto field detection
func structToMap(cfg Config) map[string]any {
	data, _ := json.Marshal(cfg)
	var m map[string]any
	_ = json.Unmarshal(data, &m)
	return m
}
