package executor

import (
	"fmt"
	"os"
	"os/exec"

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
		warningStyle.Render("  ⚠ This command will be executed with sudo privileges!"),
		promptStyle.Render("  Continue? (y/n):"),
		m.textInput.View(),
	)
}

func Execute(command string, cfg *config.Config) error {
	// show warning prompt
	p := tea.NewProgram(initialModel())
	m, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to show confirmation prompt: %w", err)
	}

	result := m.(confirmModel)
	if result.cancelled || !result.confirmed {
		fmt.Println("\n  Execution cancelled.\n")
		return nil
	}

	//fmt.Println()
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("  Executing..."))
	fmt.Println()

	// execute with sudo
	cmd := exec.Command("sudo", "sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	//fmt.Println()
	//fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("  ✓ Execution complete"))
	fmt.Println()

	return nil
}