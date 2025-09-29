package cmd

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dorochadev/oneliner/config"
	"github.com/dorochadev/oneliner/internal/executor"
	"github.com/dorochadev/oneliner/internal/llm"
	"github.com/dorochadev/oneliner/internal/prompt"
	"github.com/spf13/cobra"
)

var (
	executeFlag bool
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
	rootCmd.Flags().BoolVarP(&executeFlag, "execute", "e", false, "Execute the generated command with sudo")
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

	// parse response (command and optional explanation)
	command, explanation := parseResponse(response)

	// print output with styling
	fmt.Println()
	fmt.Println(headerStyle.Render("  Command:"))
	fmt.Println(commandStyle.Render("  " + command))

	if explainFlag && explanation != "" {
		fmt.Println()
		fmt.Println(headerStyle.Render("  Explanation:"))
		fmt.Println(explanationStyle.Render("  " + explanation))
	}
	fmt.Println()
	fmt.Println()

	// execute if requested
	if executeFlag {
		if err := executor.Execute(command, cfg); err != nil {
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

    // remove markdown code blocks
    r = strings.TrimPrefix(r, "```bash")
    r = strings.TrimPrefix(r, "```sh")
    r = strings.TrimPrefix(r, "```")
    r = strings.TrimSuffix(r, "```")
    r = strings.TrimSpace(r)

    // if there's an explanation marker
    if strings.Contains(r, "EXPLANATION:") {
        parts := strings.Split(r, "EXPLANATION:")
        command = strings.TrimSpace(parts[0])
        if len(parts) > 1 {
            explanation = strings.TrimSpace(parts[1])
        }
    } else {
        command = r
    }

    return command, explanation
}
