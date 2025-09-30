package executor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dorochadev/oneliner/config"
)

var (
	warningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("11"))

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12"))

	commandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	cancelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true)
)

type confirmModel struct {
	textInput textinput.Model
	confirmed bool
	cancelled bool
}

func initialModel() confirmModel {
	ti := textinput.New()
	ti.Placeholder = "y/n"
	ti.Focus()
	ti.CharLimit = 1
	ti.Width = 20

	return confirmModel{
		textInput: ti,
	}
}

func (m confirmModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			input := m.textInput.Value()
			if input == "y" || input == "Y" {
				m.confirmed = true
			} else {
				m.cancelled = true
			}
			return m, tea.Quit
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m confirmModel) View() string {
	return fmt.Sprintf(
		"\n%s\n\n%s %s\n\n",
		warningStyle.Render("⚠ This command will be executed with sudo privileges!"),
		promptStyle.Render("Continue? (y/n):"),
		m.textInput.View(),
	)
}

func Execute(command string, cfg *config.Config) error {
	// If the command starts with 'sudo', show sudo warning and prompt
	needsSudo := strings.HasPrefix(strings.TrimSpace(command), "sudo ")
	if needsSudo {
		p := tea.NewProgram(initialModel())
		m, err := p.Run()
		if err != nil {
			return fmt.Errorf("failed to show confirmation prompt: %w", err)
		}
		result := m.(confirmModel)
		if result.cancelled || !result.confirmed {
			fmt.Println()
			fmt.Println(cancelStyle.Render("  ✘ Execution cancelled."))
			fmt.Println()
			return nil
		}
		fmt.Println(headerStyle.Render("  Running command with sudo:"))
		fmt.Println(commandStyle.Render("  → " + command))
		// Prompt for sudo password first (to avoid blocking spinner later)
		sudoCmd := exec.Command("sudo", "-v")
		sudoCmd.Stdin = os.Stdin
		sudoCmd.Stdout = os.Stdout
		sudoCmd.Stderr = os.Stderr
		if err := sudoCmd.Run(); err != nil {
			fmt.Println()
			return fmt.Errorf("failed to authenticate with sudo: %w", err)
		}
	} else {
		fmt.Println(headerStyle.Render("  Running command:"))
		fmt.Println(commandStyle.Render("  → " + command))
	}

	// Start spinner
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Prefix = " Running command..."
	s.Start()

	// Always run the command as given
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	s.Stop()
	fmt.Println()

	if err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	fmt.Println(successStyle.Render(" ✓ Done"))
	fmt.Println()

	return nil
}
