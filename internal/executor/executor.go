package executor

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dorochadev/oneliner/config"
)

var (
	warningStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	promptStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	commandStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	cancelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	headerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
)

type confirmModel struct {
	textInput       textinput.Model
	confirmed       bool
	cancelled       bool
	showSudoWarning bool
}

func initialModel(showSudoWarning bool) confirmModel {
	ti := textinput.New()
	ti.Placeholder = "y/n"
	ti.Focus()
	ti.CharLimit = 1
	ti.Width = 20

	return confirmModel{
		textInput:       ti,
		showSudoWarning: showSudoWarning,
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
	if m.showSudoWarning {
		return fmt.Sprintf(
			"\n%s\n\n%s %s\n\n",
			warningStyle.Render("⚠ This command will be executed with sudo privileges!"),
			promptStyle.Render("Continue? (y/n):"),
			m.textInput.View(),
		)
	}
	return fmt.Sprintf(
		"%s\n\n",
		m.textInput.View(),
	)
}

func Execute(command string, cfg *config.Config, usedSudoFlag bool) error {
	trimmed := strings.TrimSpace(command)
	assessment := AssessCommandRisk(trimmed, usedSudoFlag)

	needsSudo := strings.HasPrefix(trimmed, "sudo ")
	hasRiskAssessmentIssues := len(assessment.Reasons) > 0

	// If any risks detected, show warning and ask for confirmation ONCE
	if hasRiskAssessmentIssues {
		fmt.Println()
		fmt.Println(warningStyle.Render("⚠ Dangerous command detected"))
		fmt.Println()
		fmt.Println(promptStyle.Render("The command looks potentially destructive for the following reasons:"))
		for i, r := range assessment.Reasons {
			fmt.Printf("  %d) %s\n", i+1, r)
		}
		fmt.Println()
		fmt.Println(promptStyle.Render("Command to be executed:"))
		fmt.Println(commandStyle.Render("  → " + trimmed))
		fmt.Println()
		fmt.Println(warningStyle.Render("This action can cause data loss or system damage. Continue? (y/n):"))

		// Don't show sudo warning in bubble tea if risk assessment already caught it
		p := tea.NewProgram(initialModel(false))
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

		// User confirmed - if command needs sudo, authenticate silently
		if needsSudo {
			sudoCmd := exec.Command("sudo", "-v")
			sudoCmd.Stdin = os.Stdin
			sudoCmd.Stdout = os.Stdout
			sudoCmd.Stderr = os.Stderr
			if err := sudoCmd.Run(); err != nil {
				fmt.Println()
				return fmt.Errorf("failed to authenticate with sudo: %w", err)
			}
		}

		// Show execution message and proceed directly to execution
		fmt.Println()
		fmt.Println(headerStyle.Render("  Running command:"))
		fmt.Println(commandStyle.Render("  → " + trimmed))

	} else if needsSudo && usedSudoFlag {
		// User explicitly used --sudo flag, show sudo warning
		fmt.Println()
		fmt.Println(headerStyle.Render("  Running command with sudo:"))
		fmt.Println(commandStyle.Render("  → " + trimmed))

		p := tea.NewProgram(initialModel(true))
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

		sudoCmd := exec.Command("sudo", "-v")
		sudoCmd.Stdin = os.Stdin
		sudoCmd.Stdout = os.Stdout
		sudoCmd.Stderr = os.Stderr
		if err := sudoCmd.Run(); err != nil {
			fmt.Println()
			return fmt.Errorf("failed to authenticate with sudo: %w", err)
		}

	} else if needsSudo {
		// Command has sudo but user didn't use --sudo flag and no risk issues
		// Just show we're running with sudo and authenticate
		fmt.Println()
		fmt.Println(headerStyle.Render("  Running command with sudo:"))
		fmt.Println(commandStyle.Render("  → " + trimmed))

		sudoCmd := exec.Command("sudo", "-v")
		sudoCmd.Stdin = os.Stdin
		sudoCmd.Stdout = os.Stdout
		sudoCmd.Stderr = os.Stderr
		if err := sudoCmd.Run(); err != nil {
			fmt.Println()
			return fmt.Errorf("failed to authenticate with sudo: %w", err)
		}

	} else {
		// Normal command, no risks, no sudo
		fmt.Println()
		fmt.Println(headerStyle.Render("  Running command:"))
		fmt.Println(commandStyle.Render("  → " + trimmed))
	}

	// Execute the command
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Prefix = " ..."
	s.Start()

	shell := "sh"
	args := []string{"-c", trimmed}
	if runtime.GOOS == "windows" {
		shell = "cmd"
		args = []string{"/C", trimmed}
	}

	cmd := exec.Command(shell, args...)
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
