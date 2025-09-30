package cmd

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dorochadev/oneliner/config"
	"github.com/dorochadev/oneliner/internal/cache"
	"github.com/dorochadev/oneliner/internal/executor"
	"github.com/dorochadev/oneliner/internal/llm"
	"github.com/dorochadev/oneliner/internal/prompt"
	"github.com/spf13/cobra"
)

var (
	executeFlag bool
	sudoFlag    bool
	explainFlag bool
	configPath  string

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12"))

	commandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	explanationStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("8"))
)

var rootCmd = &cobra.Command{
	Use:   "oneliner [query]",
	Short: "Generate shell one-liners from natural language",
	Long:  "A CLI tool that generates shell one-liners from natural-language input using LLMs.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  run,
}

func init() {
	rootCmd.Flags().BoolVarP(&executeFlag, "execute", "e", false, "Execute the generated command as-is")
	if runtime.GOOS != "windows" {
		rootCmd.Flags().BoolVar(&sudoFlag, "sudo", false, "Prepend 'sudo' to the generated command when executing")
	}
	rootCmd.Flags().BoolVar(&explainFlag, "explain", false, "Show an explanation of the generated command")
	rootCmd.Flags().StringVar(&configPath, "config", "", "Specify alternative config file")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// gather system context
	ctx := gatherContext(args)

	// set up cache (default: ~/.cache/oneliner/commands.json)
	home, _ := os.UserHomeDir()
	cachePath := os.Getenv("ONELINER_CACHE_PATH")
	if cachePath == "" {
		cachePath = home + "/.cache/oneliner/commands.json"
	}
	commandCache, _ := cache.New(cachePath)

	hash := cache.HashQuery(ctx.Query, ctx.OS, ctx.CWD, ctx.Username, ctx.Shell, explainFlag)
	if cached, ok := commandCache.Get(hash); ok {
		command, explanation := parseResponse(cached)
		fmt.Println(commandStyle.Render(command))
		if explainFlag && explanation != "" {
			fmt.Println(headerStyle.Render("  Explanation:"))
			fmt.Println(explanationStyle.Render("  → " + explanation))
		}
		if executeFlag {
			execCmd := command
			if runtime.GOOS == "windows" && sudoFlag {
				fmt.Fprintln(os.Stderr, "Warning: --sudo flag is not supported on Windows and will be ignored.")
			}
			if runtime.GOOS != "windows" && sudoFlag {
				execCmd = "sudo " + execCmd
			}
			if err := executor.Execute(execCmd, cfg); err != nil {
				return fmt.Errorf("failed to execute command: %w", err)
			}
		}
		return nil
	}

	// create LLM instance
	llmInstance, err := llm.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM: %w", err)
	}

	// generate prompt
	promptText := prompt.Build(ctx, cfg, explainFlag)

	// generate command
	response, err := llmInstance.GenerateCommand(promptText)
	if err != nil {
		return fmt.Errorf("failed to generate command: %w", err)
	}

	// save to cache
	_ = commandCache.Set(hash, response)

	// parse response (command and optional explanation)
	command, explanation := parseResponse(response)

	// print output with styling
	fmt.Println(commandStyle.Render(command))

	if explainFlag && explanation != "" {
		fmt.Println(headerStyle.Render("  Explanation:"))
		fmt.Println(explanationStyle.Render("  → " + explanation))
	}

	// execute if requested
	if executeFlag {
		execCmd := command
		if runtime.GOOS == "windows" && sudoFlag {
			fmt.Fprintln(os.Stderr, "Warning: --sudo flag is not supported on Windows and will be ignored.")
		}
		if runtime.GOOS != "windows" && sudoFlag {
			execCmd = "sudo " + execCmd
		}
		if err := executor.Execute(execCmd, cfg); err != nil {
			return fmt.Errorf("failed to execute command: %w", err)
		}
	}

	return nil
}

func gatherContext(args []string) prompt.Context {
	query := strings.Join(args, " ")
	cwd, _ := os.Getwd()
	u, _ := user.Current()
	username := "unknown"
	if u != nil {
		username = u.Username
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	return prompt.Context{
		Query:    query,
		OS:       runtime.GOOS,
		CWD:      cwd,
		Username: username,
		Shell:    shell,
	}
}

func parseResponse(response string) (command string, explanation string) {
	r := strings.TrimSpace(response)

	// remove any surrounding triple backticks with optional language
	r = strings.TrimPrefix(r, "```bash")
	r = strings.TrimPrefix(r, "```sh")
	r = strings.TrimPrefix(r, "```shell")
	r = strings.TrimPrefix(r, "```text")
	r = strings.TrimPrefix(r, "```")
	r = strings.TrimSuffix(r, "```")
	r = strings.TrimSpace(r)

	// split by EXPLANATION marker
	if strings.Contains(r, "EXPLANATION:") {
		parts := strings.SplitN(r, "EXPLANATION:", 2)
		command = strings.TrimSpace(parts[0])
		explanation = strings.TrimSpace(parts[1])
	} else {
		command = r
	}

	// remove any remaining backticks inside the command/explanation
	command = strings.ReplaceAll(command, "```", "")
	explanation = strings.ReplaceAll(explanation, "```", "")

	return command, explanation
}
