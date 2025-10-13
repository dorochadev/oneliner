package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	timestampStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	idStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	queryStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
)

type cacheEntryWithID struct {
	ID        string
	Command   string
	Timestamp time.Time
}

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage oneliner cache",
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all cached commands",
	RunE: func(cmd *cobra.Command, args []string) error {
		cachePath, err := getCachePath()
		if err != nil {
			return err
		}

		if _, err := os.Stat(cachePath); os.IsNotExist(err) {
			fmt.Println("Cache is already empty")
			return nil
		}

		if err := os.Remove(cachePath); err != nil {
			return fmt.Errorf("failed to clear cache: %w", err)
		}

		fmt.Println("✓ Cache cleared successfully")
		return nil
	},
}

var cacheListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all cached commands",
	RunE: func(cmd *cobra.Command, args []string) error {
		cachePath, err := getCachePath()
		if err != nil {
			return err
		}

		entries, err := loadCacheEntries(cachePath)
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			fmt.Println("Cache is empty")
			return nil
		}

		// Sort by timestamp, newest first
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Timestamp.After(entries[j].Timestamp)
		})

		fmt.Printf("Found %d cached command(s):\n\n", len(entries))

		for _, entry := range entries {
			// Truncate ID for display
			shortID := entry.ID[:min(8, len(entry.ID))]

			// Parse command and explanation
			command, explanation, _ := parseResponse(entry.Command)

			// Truncate command if too long
			displayCmd := command
			if len(displayCmd) > 80 {
				displayCmd = displayCmd[:77] + "..."
			}

			// Format timestamp
			timeStr := formatTimestamp(entry.Timestamp)

			fmt.Printf("%s %s\n",
				idStyle.Render(shortID),
				queryStyle.Render(displayCmd))

			if explanation != "" {
				explainPreview := explanation
				if len(explainPreview) > 80 {
					explainPreview = explainPreview[:77] + "..."
				}
				fmt.Printf("    %s\n", dimStyle.Render(explainPreview))
			}

			fmt.Printf("    %s\n\n", timestampStyle.Render(timeStr))
		}

		fmt.Printf("Use 'oneliner cache rm <id>' to remove a specific entry\n")
		fmt.Printf("Use 'oneliner cache clear' to clear all entries\n")

		return nil
	},
}

var cacheRmCmd = &cobra.Command{
	Use:   "rm [id]",
	Short: "Remove a cached command by ID (prefix)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		idPrefix := args[0]

		cachePath, err := getCachePath()
		if err != nil {
			return err
		}

		entries, err := loadCacheEntries(cachePath)
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			return fmt.Errorf("cache is empty")
		}

		// Find matching ID(s)
		var matchedID string
		matchCount := 0
		for _, entry := range entries {
			if strings.HasPrefix(entry.ID, idPrefix) {
				matchedID = entry.ID
				matchCount++
			}
		}

		if matchCount == 0 {
			return fmt.Errorf("no cached entry found with ID prefix: %s", idPrefix)
		}

		if matchCount > 1 {
			return fmt.Errorf("ambiguous ID prefix '%s' matches %d entries, please be more specific", idPrefix, matchCount)
		}

		// Remove the entry from cache
		if err := deleteCacheEntry(cachePath, matchedID); err != nil {
			return fmt.Errorf("failed to remove entry: %w", err)
		}

		fmt.Printf("✓ Removed cached entry: %s\n", idStyle.Render(matchedID[:min(8, len(matchedID))]))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheClearCmd)
	cacheCmd.AddCommand(cacheListCmd)
	cacheCmd.AddCommand(cacheRmCmd)
}

func getCachePath() (string, error) {
	cachePath := os.Getenv("ONELINER_CACHE_PATH")
	if cachePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		cachePath = filepath.Join(home, ".cache", "oneliner", "commands.json")
	}

	absPath, err := filepath.Abs(cachePath)
	if err != nil {
		return "", fmt.Errorf("invalid cache path: %w", err)
	}

	if !strings.HasSuffix(absPath, ".json") {
		return "", fmt.Errorf("cache path must be a .json file")
	}

	return absPath, nil
}

func loadCacheEntries(cachePath string) ([]cacheEntryWithID, error) {
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return []cacheEntryWithID{}, nil
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	// Try new format first
	var cacheData map[string]struct {
		Command   string    `json:"command"`
		Timestamp time.Time `json:"timestamp"`
	}

	if err := json.Unmarshal(data, &cacheData); err != nil {
		// Try legacy format
		var legacyData map[string]string
		if err := json.Unmarshal(data, &legacyData); err != nil {
			return nil, fmt.Errorf("failed to parse cache file: %w", err)
		}

		// Convert legacy format
		entries := make([]cacheEntryWithID, 0, len(legacyData))
		for id, cmd := range legacyData {
			entries = append(entries, cacheEntryWithID{
				ID:        id,
				Command:   cmd,
				Timestamp: time.Time{}, // unknown timestamp for legacy
			})
		}
		return entries, nil
	}

	entries := make([]cacheEntryWithID, 0, len(cacheData))
	for id, entry := range cacheData {
		entries = append(entries, cacheEntryWithID{
			ID:        id,
			Command:   entry.Command,
			Timestamp: entry.Timestamp,
		})
	}

	return entries, nil
}

func deleteCacheEntry(cachePath string, idToRemove string) error {
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	var cacheData map[string]struct {
		Command   string    `json:"command"`
		Timestamp time.Time `json:"timestamp"`
	}

	if err := json.Unmarshal(data, &cacheData); err != nil {
		// Try legacy format
		var legacyData map[string]string
		if err := json.Unmarshal(data, &legacyData); err != nil {
			return fmt.Errorf("failed to parse cache file: %w", err)
		}

		delete(legacyData, idToRemove)

		newData, err := json.MarshalIndent(legacyData, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal cache: %w", err)
		}

		return os.WriteFile(cachePath, newData, 0600)
	}

	delete(cacheData, idToRemove)

	newData, err := json.MarshalIndent(cacheData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	return os.WriteFile(cachePath, newData, 0600)
}

func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}

	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("2006-01-02")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
