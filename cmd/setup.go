package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dorochadev/oneliner/config"
	"github.com/spf13/cobra"
)

var (
	cursorStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	titleStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	subtitleStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	selectedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	unselectedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	setupCancelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	setupSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	setupHintStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type setupModel struct {
	step             int
	selectedAPI      int
	inputs           []textinput.Model
	cfg              *config.Config
	cfgPath          string
	done             bool
	cancelled        bool
	apiOptions       []string
	modelSuggestions map[string][]string
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard for oneliner configuration",
	Long:  "Guide you through configuring your LLM provider, API key, and other settings.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath := ""
		if cfgPath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home dir: %w", err)
			}
			cfgPath = filepath.Join(home, ".config", "oneliner", "config.json")
		}

		// Load existing config or create default
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		p := tea.NewProgram(initialSetupModel(cfg, cfgPath))
		m, err := p.Run()
		if err != nil {
			return fmt.Errorf("failed to run setup: %w", err)
		}

		result := m.(setupModel)
		if result.cancelled {
			fmt.Println(setupCancelStyle.Render("\n  ✗ Setup cancelled\n"))
			return nil
		}

		if result.done {
			fmt.Println(setupSuccessStyle.Render("\n  ✓ Configuration saved successfully!\n"))
			fmt.Println(setupHintStyle.Render("  Try it out:"))
			fmt.Println(setupHintStyle.Render("    oneliner \"list all files larger than 10MB\"\n"))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func initialSetupModel(cfg *config.Config, cfgPath string) setupModel {
	apiOptions := []string{"openai", "claude", "local"}

	modelSuggestions := map[string][]string{
		"openai": {"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo"},
		"claude": {"claude-sonnet-4-5-20250929", "claude-3-5-sonnet-20241022", "claude-3-opus-20240229"},
		"local":  {"llama3", "mistral", "codellama"},
	}

	// Create text inputs for configuration
	inputs := make([]textinput.Model, 4)

	// API Key input
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "sk-..."
	inputs[0].CharLimit = 200
	inputs[0].Width = 50
	inputs[0].EchoMode = textinput.EchoPassword
	inputs[0].EchoCharacter = '•'

	// Model input
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "gpt-4o"
	inputs[1].CharLimit = 100
	inputs[1].Width = 50

	// Local endpoint input
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "http://localhost:8000/v1/completions"
	inputs[2].CharLimit = 200
	inputs[2].Width = 50

	// Max tokens input
	inputs[3] = textinput.New()
	inputs[3].Placeholder = "1024"
	inputs[3].CharLimit = 10
	inputs[3].Width = 20

	// Pre-fill with existing values
	selectedAPI := 0
	for i, opt := range apiOptions {
		if opt == cfg.LLMAPI {
			selectedAPI = i
			break
		}
	}

	if cfg.APIKey != "" {
		inputs[0].SetValue(cfg.APIKey)
	}
	if cfg.Model != "" {
		inputs[1].SetValue(cfg.Model)
	}
	if cfg.LocalLLMEndpoint != "" {
		inputs[2].SetValue(cfg.LocalLLMEndpoint)
	}
	if cfg.ClaudeMaxTokens > 0 {
		inputs[3].SetValue(fmt.Sprintf("%d", cfg.ClaudeMaxTokens))
	}

	return setupModel{
		step:             0,
		selectedAPI:      selectedAPI,
		inputs:           inputs,
		cfg:              cfg,
		cfgPath:          cfgPath,
		apiOptions:       apiOptions,
		modelSuggestions: modelSuggestions,
	}
}

func (m setupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m setupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit

		case "enter":
			return m.handleEnter()

		case "up", "k":
			if m.step == 0 && m.selectedAPI > 0 {
				m.selectedAPI--
			}

		case "down", "j":
			if m.step == 0 && m.selectedAPI < len(m.apiOptions)-1 {
				m.selectedAPI++
			}

		case "tab", "shift+tab":
			// Skip to next/previous relevant input based on API selection
			if m.step > 0 {
				return m.handleTab(msg.String() == "shift+tab")
			}
		}
	}

	// Update the current input if we're past API selection
	if m.step > 0 {
		var cmd tea.Cmd
		inputIdx := m.getInputIndex()
		if inputIdx >= 0 && inputIdx < len(m.inputs) {
			m.inputs[inputIdx], cmd = m.inputs[inputIdx].Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m setupModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.step {
	case 0:
		// API selection confirmed, move to next step
		m.cfg.LLMAPI = m.apiOptions[m.selectedAPI]
		m.step++
		m.inputs[m.getInputIndex()].Focus()
		return m, textinput.Blink

	case 1, 2, 3, 4:
		// Save current input and move to next
		if err := m.saveCurrentStep(); err != nil {
			// Handle error (for now just continue)
			return m, nil
		}

		if m.isLastStep() {
			// Save configuration
			if err := m.saveConfig(); err != nil {
				m.cancelled = true
				return m, tea.Quit
			}
			m.done = true
			return m, tea.Quit
		}

		m.step++
		nextInput := m.getInputIndex()
		if nextInput >= 0 && nextInput < len(m.inputs) {
			m.inputs[nextInput].Focus()
		}
		return m, textinput.Blink
	}

	return m, nil
}

func (m setupModel) handleTab(reverse bool) (tea.Model, tea.Cmd) {
	currentIdx := m.getInputIndex()
	if currentIdx < 0 {
		return m, nil
	}

	m.inputs[currentIdx].Blur()

	if reverse {
		m.step--
		if m.step <= 0 {
			m.step = 1
		}
	} else {
		if !m.isLastStep() {
			m.step++
		}
	}

	nextIdx := m.getInputIndex()
	if nextIdx >= 0 && nextIdx < len(m.inputs) {
		m.inputs[nextIdx].Focus()
	}

	return m, textinput.Blink
}

func (m *setupModel) saveCurrentStep() error {
	apiType := m.cfg.LLMAPI

	switch m.step {
	case 1: // API Key or Local Endpoint (depending on API)
		if apiType == "local" {
			m.cfg.LocalLLMEndpoint = strings.TrimSpace(m.inputs[2].Value())
		} else {
			m.cfg.APIKey = strings.TrimSpace(m.inputs[0].Value())
		}
	case 2: // Model
		m.cfg.Model = strings.TrimSpace(m.inputs[1].Value())
	case 3: // Max tokens (for Claude)
		if apiType == "claude" {
			val := strings.TrimSpace(m.inputs[3].Value())
			if val != "" {
				var tokens int
				fmt.Sscanf(val, "%d", &tokens)
				if tokens > 0 {
					m.cfg.ClaudeMaxTokens = tokens
				}
			}
		}
	}

	return nil
}

func (m setupModel) isLastStep() bool {
	apiType := m.cfg.LLMAPI

	switch apiType {
	case "local":
		return m.step >= 2 // endpoint + model
	case "claude":
		return m.step >= 3 // api key + model + max tokens
	case "openai":
		return m.step >= 2 // api key + model
	default:
		return m.step >= 2
	}
}

func (m setupModel) getInputIndex() int {
	apiType := m.cfg.LLMAPI

	if m.step == 1 {
		if apiType == "local" {
			return 2 // local endpoint
		}
		return 0 // api key
	}

	if m.step == 2 {
		return 1 // model
	}

	if m.step == 3 && apiType == "claude" {
		return 3 // max tokens
	}

	return -1
}

func (m *setupModel) saveConfig() error {
	// Set defaults if empty
	if m.cfg.Model == "" {
		suggestions := m.modelSuggestions[m.cfg.LLMAPI]
		if len(suggestions) > 0 {
			m.cfg.Model = suggestions[0]
		}
	}

	if m.cfg.ClaudeMaxTokens <= 0 {
		m.cfg.ClaudeMaxTokens = 1024
	}

	if m.cfg.RequestTimeout <= 0 {
		m.cfg.RequestTimeout = 60
	}

	if m.cfg.ClientTimeout <= 0 {
		m.cfg.ClientTimeout = 65
	}

	// Save to file
	return config.Save(m.cfgPath, m.cfg)
}

func (m setupModel) View() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(titleStyle.Render("  ⚙️  oneliner setup"))
	b.WriteString("\n\n")

	switch m.step {
	case 0:
		return m.viewAPISelection()
	case 1:
		return m.viewCredentials()
	case 2:
		return m.viewModel()
	case 3:
		if m.cfg.LLMAPI == "claude" {
			return m.viewMaxTokens()
		}
	}

	return b.String()
}

func (m setupModel) viewAPISelection() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(titleStyle.Render("  ⚙️  oneliner setup"))
	b.WriteString("\n\n")
	b.WriteString(subtitleStyle.Render("  Select your LLM provider:"))
	b.WriteString("\n\n")

	for i, option := range m.apiOptions {
		cursor := "  "
		if i == m.selectedAPI {
			cursor = cursorStyle.Render("▸ ")
			b.WriteString(cursor)
			b.WriteString(selectedStyle.Render(option))
		} else {
			b.WriteString(cursor)
			b.WriteString(unselectedStyle.Render(option))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(hintStyle.Render("  ↑/↓ navigate • enter confirm • esc cancel"))
	b.WriteString("\n")

	return b.String()
}

func (m setupModel) viewCredentials() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(titleStyle.Render("  ⚙️  oneliner setup"))
	b.WriteString("\n\n")

	apiType := m.cfg.LLMAPI

	if apiType == "local" {
		b.WriteString(subtitleStyle.Render("  Local LLM Endpoint:"))
		b.WriteString("\n\n")
		b.WriteString("  ")
		b.WriteString(m.inputs[2].View())
		b.WriteString("\n\n")
		b.WriteString(hintStyle.Render("  Example: http://localhost:8000/v1/completions"))
	} else {
		b.WriteString(subtitleStyle.Render(fmt.Sprintf("  %s API Key:", strings.ToUpper(apiType))))
		b.WriteString("\n\n")
		b.WriteString("  ")
		b.WriteString(m.inputs[0].View())
		b.WriteString("\n\n")

		if apiType == "openai" {
			b.WriteString(hintStyle.Render("  Get your key: https://platform.openai.com/api-keys"))
		} else if apiType == "claude" {
			b.WriteString(hintStyle.Render("  Get your key: https://console.anthropic.com/"))
		}
	}

	b.WriteString("\n\n")
	b.WriteString(hintStyle.Render("  enter continue • tab navigate • esc cancel"))
	b.WriteString("\n")

	return b.String()
}

func (m setupModel) viewModel() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(titleStyle.Render("  ⚙️  oneliner setup"))
	b.WriteString("\n\n")
	b.WriteString(subtitleStyle.Render("  Model name:"))
	b.WriteString("\n\n")
	b.WriteString("  ")
	b.WriteString(m.inputs[1].View())
	b.WriteString("\n\n")

	suggestions := m.modelSuggestions[m.cfg.LLMAPI]
	if len(suggestions) > 0 {
		b.WriteString(hintStyle.Render("  Suggestions: " + strings.Join(suggestions, ", ")))
	}

	b.WriteString("\n\n")
	b.WriteString(hintStyle.Render("  enter continue • tab navigate • esc cancel"))
	b.WriteString("\n")

	return b.String()
}

func (m setupModel) viewMaxTokens() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(titleStyle.Render("  ⚙️  oneliner setup"))
	b.WriteString("\n\n")
	b.WriteString(subtitleStyle.Render("  Max tokens (Claude):"))
	b.WriteString("\n\n")
	b.WriteString("  ")
	b.WriteString(m.inputs[3].View())
	b.WriteString("\n\n")
	b.WriteString(hintStyle.Render("  Recommended: 1024-4096"))
	b.WriteString("\n\n")
	b.WriteString(hintStyle.Render("  enter finish • tab navigate • esc cancel"))
	b.WriteString("\n")

	return b.String()
}
