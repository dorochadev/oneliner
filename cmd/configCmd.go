package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/dorochadev/oneliner/config"
	"github.com/spf13/cobra"
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
					switch fieldVal.Kind() {
					case reflect.String:
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

		fmt.Printf("Configuration updated: %s = %s\n", key, value)
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

		fmt.Println("Current configuration:")
		fmt.Printf("%-20s %-20s %-10s\n", "KEY", "VALUE", "TYPE")
		fmt.Println(strings.Repeat("-", 55))

		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			jsonTag := field.Tag.Get("json")
			fieldVal := v.Field(i)

			value := ""
			switch fieldVal.Kind() {
			case reflect.String:
				value = fieldVal.String()
			case reflect.Int:
				value = strconv.Itoa(int(fieldVal.Int()))
			default:
				value = "<unsupported type>"
			}

			fmt.Printf("%-20s %-20s %-10s\n", jsonTag, value, fieldVal.Kind().String())
		}

		fmt.Println("\nUse `oneliner config set <KEY> <VALUE>` to update a setting.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(setCmd)
	configCmd.AddCommand(listCmd)
}
