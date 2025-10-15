package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dorochadev/oneliner/config"
	"github.com/spf13/cobra"
)

var (
	keyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	valueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	typeStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	headerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Bold(true)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	hintStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage oneliner configuration",
}

var setCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		cfgPath := "" // use default
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Store old value for display
		oldValue := ""

		// reflect over Config struct to set field dynamically
		v := reflect.ValueOf(cfg).Elem()
		t := v.Type()
		found := false
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			jsonTag := field.Tag.Get("json")
			if jsonTag == key {
				fieldVal := v.FieldByName(field.Name)
				if fieldVal.CanSet() {
					// Store old value
					switch fieldVal.Kind() {
					case reflect.String:
						oldValue = fieldVal.String()
					case reflect.Int:
						oldValue = strconv.Itoa(int(fieldVal.Int()))
					}

					// Set new value
					switch fieldVal.Kind() {
					case reflect.String:
						if jsonTag == "local_llm_endpoint" && value != "" {
							if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
								return fmt.Errorf("endpoint must start with http:// or https://")
							}
						}
						fieldVal.SetString(value)
					case reflect.Int:
						var intVal int
						_, err := fmt.Sscanf(value, "%d", &intVal)
						if err != nil {
							return fmt.Errorf("invalid integer value for %s: %v", key, err)
						}
						fieldVal.SetInt(int64(intVal))
					default:
						return fmt.Errorf("unsupported field type for %s", key)
					}
					found = true
					break
				}
			}
		}

		if !found {
			return fmt.Errorf("unknown config key: %s", key)
		}

		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to serialize config: %w", err)
		}

		if cfgPath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home dir: %w", err)
			}
			cfgPath = filepath.Join(home, ".config", "oneliner", "config.json")
		}

		if err := os.WriteFile(cfgPath, data, 0600); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		fmt.Println()
		fmt.Print(successStyle.Render("  ✓ Configuration updated"))
		fmt.Println()
		fmt.Println()

		// Show the change
		fmt.Printf("  %s\n", keyStyle.Render(key))
		if oldValue != "" && oldValue != value {
			fmt.Printf("    %s → %s\n", hintStyle.Render(oldValue), valueStyle.Render(value))
		} else {
			fmt.Printf("    %s\n", valueStyle.Render(value))
		}
		fmt.Println()

		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List current configuration values",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath := ""
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		v := reflect.ValueOf(cfg).Elem()
		t := v.Type()

		fmt.Println()
		fmt.Println(headerStyle.Render("  Configuration"))
		fmt.Println()

		// Find longest key for alignment
		maxKeyLen := 0
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			jsonTag := field.Tag.Get("json")
			if len(jsonTag) > maxKeyLen {
				maxKeyLen = len(jsonTag)
			}
		}

		// Print each config entry with better formatting
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			jsonTag := field.Tag.Get("json")
			fieldVal := v.Field(i)

			value := ""
			typeStr := ""
			switch fieldVal.Kind() {
			case reflect.String:
				value = fieldVal.String()
				if value == "" {
					value = hintStyle.Render("<not set>")
				} else if jsonTag == "api_key" && value != "" {
					// Mask API key
					if len(value) > 8 {
						value = value[:4] + "..." + value[len(value)-4:]
					} else {
						value = "***"
					}
					value = valueStyle.Render(value)
				} else {
					value = valueStyle.Render(value)
				}
				typeStr = "string"
			case reflect.Int:
				value = valueStyle.Render(strconv.Itoa(int(fieldVal.Int())))
				typeStr = "int"

			case reflect.Slice:
				// handle []string gracefully
				if fieldVal.Len() == 0 {
					value = hintStyle.Render("[]")
				} else {
					elems := make([]string, fieldVal.Len())
					for j := 0; j < fieldVal.Len(); j++ {
						elem := fieldVal.Index(j)
						elems[j] = fmt.Sprintf("%v", elem.Interface())
					}
					joined := "[" + strings.Join(elems, ", ") + "]"
					value = valueStyle.Render(joined)
				}
				typeStr = "array[string]"

			default:
				value = hintStyle.Render("<unsupported>")
				typeStr = fieldVal.Kind().String()
			}

			// Format: key (type) : value
			padding := strings.Repeat(" ", maxKeyLen-len(jsonTag))
			fmt.Printf("  %s%s %s %s\n",
				keyStyle.Render(jsonTag),
				padding,
				typeStyle.Render(fmt.Sprintf("(%s)", typeStr)),
				value)
		}

		fmt.Println()
		fmt.Println(hintStyle.Render("  Use 'oneliner config set <key> <value>' to update"))
		fmt.Println()

		return nil
	},
}

var openCmd = &cobra.Command{
	Use:   "open",
	Short: "Open the default config in your editor",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath := ""
		if _, err := config.Load(cfgPath); err != nil {
			return fmt.Errorf("failed to ensure config exists: %w", err)
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home dir: %w", err)
		}
		cfgPath = filepath.Join(home, ".config", "oneliner", "config.json")

		editor := os.Getenv("EDITOR")
		if editor != "" {
			c := exec.Command(editor, cfgPath)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Run(); err != nil {
				return fmt.Errorf("failed to open config with $EDITOR: %w", err)
			}
		} else {
			switch runtime.GOOS {
			case "windows":
				c := exec.Command("cmd", "/C", "start", "", cfgPath)
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				if err := c.Run(); err != nil {
					return fmt.Errorf("failed to open config: %w", err)
				}
			case "darwin":
				if err := exec.Command("open", cfgPath).Start(); err != nil {
					return fmt.Errorf("failed to open config: %w", err)
				}
			default:
				if err := exec.Command("xdg-open", cfgPath).Start(); err != nil {
					return fmt.Errorf("failed to open config: %w", err)
				}
			}
		}

		fmt.Println()
		fmt.Print(successStyle.Render("  ✓ Opened config"))
		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(setCmd)
	configCmd.AddCommand(listCmd)
	configCmd.AddCommand(openCmd)
}
