package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/briandowns/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dorochadev/oneliner/config"
	"github.com/dorochadev/oneliner/internal/cache"
	"github.com/dorochadev/oneliner/internal/executor"
	"github.com/dorochadev/oneliner/internal/llm"
	"github.com/dorochadev/oneliner/internal/prompt"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	executeFlag      bool
	interactiveFlag  bool
	sudoFlag         bool
	explainFlag      bool
	breakdownFlag    bool
	configPath       string
	clipboardFlag    bool
	commandStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	explanationStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	breakdownStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	dimStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	cancelStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	cyanStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	rng              = rand.New(rand.NewSource(time.Now().UnixNano()))
)

var loadingMessages = []string{
	"⚙️ Generating one-liner...",
	"🔍 Finding the simplest command...",
	"🧠 Thinking through your request...",
	"💡 Turning your idea into code...",
	"🔧 Assembling the perfect command...",
	"🌊 Mapping intent to shell syntax...",
	"🧩 Piecing together your request...",
	"📦 Packing it all into one clean line...",
	"🪄 Translating thoughts into terminal language...",
}

func randomLoadingMessage() string {
	return loadingMessages[rng.Intn(len(loadingMessages))]
}

var rootCmd = &cobra.Command{
	Use:   "oneliner [query]",
	Short: "Generate shell one-liners from natural language",
	Long:  "A CLI tool that generates shell one-liners from natural-language input using LLMs.",
	Args:  cobra.MinimumNArgs(1),
	RunE:  run,
}

func init() {
	rootCmd.Flags().BoolVarP(&executeFlag, "run", "r", false, "Run the generated command as-is")
	if runtime.GOOS != "windows" {
		rootCmd.Flags().BoolVar(&sudoFlag, "sudo", false, "Prepend 'sudo' to the generated command when executing")
	}
	rootCmd.Flags().BoolVarP(&explainFlag, "explain", "e", false, "Show an explanation of the generated command")
	rootCmd.Flags().BoolVarP(&breakdownFlag, "breakdown", "b", false, "Include a detailed breakdown/pipeline of how the command works")
	rootCmd.Flags().BoolVarP(&interactiveFlag, "interactive", "i", false, "Interactively run the generated command")
	rootCmd.Flags().StringVar(&configPath, "config", "", "Specify alternative config file")
	rootCmd.Flags().BoolVarP(&clipboardFlag, "clipboard", "c", false, "Copy the generated command to clipboard")
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

	// set up cache
	commandCache, err := setupCache()
	if err != nil {
		return fmt.Errorf("failed to setup cache: %w", err)
	}

	hash := cache.HashQuery(ctx.Query, ctx.OS, ctx.CWD, ctx.Username, ctx.Shell, explainFlag, breakdownFlag)
	if cached, ok := commandCache.Get(hash); ok {
		return handleCachedCommand(cached, cfg)
	}

	// create LLM instance
	llmInstance, err := llm.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM: %w", err)
	}

	// generate prompt
	promptText, err := prompt.Build(ctx, cfg, explainFlag, breakdownFlag)
	if err != nil {
		return fmt.Errorf("failed to build prompt: %w", err)
	}

	response, err := generateWithSpinner(llmInstance, promptText)
	if err != nil {
		return fmt.Errorf("failed to generate command: %w", err)
	}

	// save to cache
	if err := commandCache.Set(hash, response); err != nil {
		return fmt.Errorf("warning: failed to write to cache: %v", err)
	}

	return handleGeneratedCommand(response, cfg)
}

func setupCache() (*cache.Cache, error) {
	cachePath := os.Getenv("ONELINER_CACHE_PATH")
	if cachePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		cachePath = filepath.Join(home, ".cache", "oneliner", "commands.json")
	}
	return cache.New(cachePath)
}

func generateWithSpinner(llmInstance llm.LLM, promptText string) (string, error) {
	loadingMsg := randomLoadingMessage()
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Prefix = loadingMsg + " "
	s.Start()
	defer func() {
		s.Stop()
		fmt.Print("\r\033[K")
	}()

	return llmInstance.GenerateCommand(promptText)
}

func handleCachedCommand(cached string, cfg *config.Config) error {
	command, explanation, breakdown := parseResponse(cached)
	displayCommand(command, explanation, breakdown)

	if clipboardFlag {
		if err := copyToClipboard(command); err != nil {
			fmt.Fprintln(os.Stderr, "Failed to copy to clipboard:", err)
		}
	}

	if executeFlag {
		return executeCommand(command, cfg)
	}

	if interactiveFlag {
		execute := displayInteractiveCommand(command, cfg)
		if execute {
			return executeCommand(command, cfg)
		}
	}

	return nil
}

func handleGeneratedCommand(response string, cfg *config.Config) error {
	command, explanation, breakdown := parseResponse(response)
	displayCommand(command, explanation, breakdown)

	if clipboardFlag {
		if err := copyToClipboard(command); err != nil {
			fmt.Fprintln(os.Stderr, "Failed to copy to clipboard:", err)
		}
	}

	if executeFlag {
		return executeCommand(command, cfg)
	}

	if interactiveFlag {
		execute := displayInteractiveCommand(command, cfg)
		if execute {
			return executeCommand(command, cfg)
		}
	}

	return nil
}

func displayCommand(command, explanation, breakdown string) {
	fmt.Println(commandStyle.Render(command))

	width := 80
	if fd := int(os.Stdout.Fd()); term.IsTerminal(fd) {
		if w, _, err := term.GetSize(fd); err == nil && w > 0 {
			width = w
		}
	}

	contentWidth := width - 6
	if contentWidth < 40 {
		contentWidth = 40
	}

	textBoxStyle := lipgloss.NewStyle().
		Width(contentWidth).
		PaddingLeft(4).
		PaddingRight(2).
		Foreground(lipgloss.Color("8"))

	headingStyle := dimStyle.Bold(true)

	if explainFlag && explanation != "" {
		fmt.Println(dimStyle.Render("  ───────────────────────────────────────"))
		fmt.Print(dimStyle.Render("  ℹ "))
		fmt.Println(headingStyle.Render("Explanation:"))
		fmt.Println(textBoxStyle.Render(explanation))
		fmt.Println()
	}

	if breakdownFlag && breakdown != "" {
		fmt.Println(dimStyle.Render("  ───────────────────────────────────────"))
		fmt.Print(dimStyle.Render("  ⤷ "))
		fmt.Println(headingStyle.Render("Breakdown:"))
		fmt.Println(textBoxStyle.Render(breakdown))
		fmt.Println()
	}
}

func displayInteractiveCommand(_ string, _ *config.Config) bool {
	fmt.Println()
	fmt.Print(cyanStyle.Render("Run command? [y/N]"))
	fmt.Println()

	p := tea.NewProgram(executor.InterationModel("", "", false))
	m, err := p.Run()
	if err != nil {
		return false
	}
	result := m.(executor.InteractionModel)
	if result.Cancelled || !result.Confirmed {
		fmt.Print(cancelStyle.Render("  ✗ CANCELLED"))
		fmt.Print(" ")
		fmt.Println(dimStyle.Render("• user aborted"))
		fmt.Println()
		return false
	}

	return true
}

func executeCommand(command string, cfg *config.Config) error {
	execCmd := command

	if runtime.GOOS == "windows" && sudoFlag {
		fmt.Fprintln(os.Stderr, "Warning: --sudo flag is not supported on Windows and will be ignored.")
	} else if runtime.GOOS != "windows" && sudoFlag {
		execCmd = "sudo " + execCmd
	}

	if err := executor.Execute(execCmd, cfg, sudoFlag); err != nil {
		return fmt.Errorf("failed to run command: %w", err)
	}
	return nil
}

func detectShell() string {
	if runtime.GOOS == "windows" {
		comspec := os.Getenv("ComSpec")
		if strings.Contains(strings.ToLower(comspec), "cmd.exe") {
			return "cmd"
		}

		if pwsh := os.Getenv("PSModulePath"); pwsh != "" {
			return "powershell"
		}

		if os.Getenv("WSL_DISTRO_NAME") != "" {
			return "bash"
		}

		return "powershell"
	}

	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	return shell
}

func gatherContext(args []string) prompt.Context {
	query := strings.Join(args, " ")
	cwd, _ := os.Getwd()
	u, _ := user.Current()
	username := "unknown"
	if u != nil {
		username = u.Username
	}

	shell := detectShell()

	return prompt.Context{
		Query:    query,
		OS:       runtime.GOOS,
		CWD:      cwd,
		Username: username,
		Shell:    shell,
	}
}

func parseResponse(response string) (command string, explanation string, breakdown string) {
	r := strings.TrimSpace(response)

	r = strings.TrimPrefix(r, "```bash")
	r = strings.TrimPrefix(r, "```sh")
	r = strings.TrimPrefix(r, "```shell")
	r = strings.TrimPrefix(r, "```powershell")
	r = strings.TrimPrefix(r, "```text")
	r = strings.TrimPrefix(r, "```")
	r = strings.TrimSuffix(r, "```")
	r = strings.TrimSpace(r)

	idxExp := strings.Index(r, "EXPLANATION:")
	idxBrk := strings.Index(r, "BREAKDOWN:")

	if idxExp == -1 && idxBrk == -1 {
		command = strings.ReplaceAll(r, "```", "")
		return command, "", ""
	}

	firstIdx := -1
	if idxExp >= 0 && (idxBrk == -1 || idxExp < idxBrk) {
		firstIdx = idxExp
	} else {
		firstIdx = idxBrk
	}

	command = strings.TrimSpace(r[:firstIdx])

	rest := strings.TrimSpace(r[firstIdx:])

	expPart := ""
	brkPart := ""

	if strings.Contains(rest, "EXPLANATION:") && strings.Contains(rest, "BREAKDOWN:") {
		expIdx := strings.Index(rest, "EXPLANATION:")
		brkIdx := strings.Index(rest, "BREAKDOWN:")
		if expIdx < brkIdx {
			expPart = strings.TrimSpace(rest[expIdx+len("EXPLANATION:") : brkIdx])
			brkPart = strings.TrimSpace(rest[brkIdx+len("BREAKDOWN:"):])
		} else {
			brkPart = strings.TrimSpace(rest[brkIdx+len("BREAKDOWN:") : expIdx])
			expPart = strings.TrimSpace(rest[expIdx+len("EXPLANATION:"):])
		}
	} else if strings.Contains(rest, "EXPLANATION:") {
		expPart = strings.TrimSpace(strings.TrimPrefix(rest, "EXPLANATION:"))
	} else if strings.Contains(rest, "BREAKDOWN:") {
		brkPart = strings.TrimSpace(strings.TrimPrefix(rest, "BREAKDOWN:"))
	}

	command = strings.ReplaceAll(command, "```", "")
	expPart = strings.ReplaceAll(expPart, "```", "")
	brkPart = strings.ReplaceAll(brkPart, "```", "")

	return command, expPart, brkPart
}

func copyToClipboard(command string) error {
	return clipboard.WriteAll(command)
}
