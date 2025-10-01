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
	warningStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	promptStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	commandStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	cancelStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	successStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	headerStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	dimStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	whiteStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	cyanStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	tagStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true).
				Background(lipgloss.Color("0")).Padding(0, 1)
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
			"\n%s %s\n\n%s %s\n\n",
			warningStyle.Render("⚠"),
			whiteStyle.Render("Requires elevated privileges"),
			dimStyle.Render("Proceed?"),
			cyanStyle.Render("[y/N]"),
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
		fmt.Print(warningStyle.Render("⚠"))
		fmt.Print(" ")
		fmt.Println(whiteStyle.Render("Command requires caution"))
		fmt.Println()
		
		// Print box top
		fmt.Println(dimStyle.Render("  ┌─────────────────────────────────────────"))
		
		// Print risks
		for i, r := range assessment.Reasons {
			fmt.Printf("%s %d) %s\n", dimStyle.Render("  │"), i+1, dimStyle.Render(r))
		}
		
		// Print command
		fmt.Println(dimStyle.Render("  │"))
		fmt.Print(dimStyle.Render("  │ "))
		fmt.Print(cyanStyle.Render("❯"))
		fmt.Print(" ")
		fmt.Println(commandStyle.Render(trimmed))
		
		// Print box bottom
		fmt.Println(dimStyle.Render("  └─────────────────────────────────────────"))
		fmt.Println()
		
		fmt.Print(dimStyle.Render("  Proceed? "))
		fmt.Print(cyanStyle.Render("[y/N] "))
	
		// Don't show sudo warning in bubble tea if risk assessment already caught it
		p := tea.NewProgram(initialModel(false))
		m, err := p.Run()
		if err != nil {
			return fmt.Errorf("failed to show confirmation prompt: %w", err)
		}
		result := m.(confirmModel)
		if result.cancelled || !result.confirmed {
			fmt.Println()
			fmt.Print(cancelStyle.Render("  ✗ CANCELLED"))
			fmt.Print(" ")
			fmt.Println(dimStyle.Render("• user aborted"))
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

		fmt.Println()
		fmt.Print(dimStyle.Render("  "))
		fmt.Print(tagStyle.Render(" sudo "))
		fmt.Print(" ")
		fmt.Print(cyanStyle.Render("❯"))
		fmt.Print(" ")
		fmt.Println(whiteStyle.Render(trimmed))

	} else if needsSudo && usedSudoFlag {
		// User explicitly used --sudo flag, show sudo warning
		fmt.Println()
		fmt.Print(warningStyle.Render("⚠"))
		fmt.Print(" ")
		fmt.Print(whiteStyle.Render("Requires elevated privileges"))
		fmt.Print(" ")
		fmt.Println(tagStyle.Render(" SUDO "))
		fmt.Println()

		p := tea.NewProgram(initialModel(true))
		m, err := p.Run()
		if err != nil {
			return fmt.Errorf("failed to show confirmation prompt: %w", err)
		}
		result := m.(confirmModel)
		if result.cancelled || !result.confirmed {
			fmt.Println()
			fmt.Print(cancelStyle.Render("  ✗ CANCELLED"))
			fmt.Print(" ")
			fmt.Println(dimStyle.Render("• user aborted"))
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

		fmt.Println()
		fmt.Print(dimStyle.Render("  "))
		fmt.Print(tagStyle.Render(" sudo "))
		fmt.Print(" ")
		fmt.Print(cyanStyle.Render("❯"))
		fmt.Print(" ")
		fmt.Println(whiteStyle.Render(trimmed))

		} else if needsSudo {
			// Command has sudo but user didn't use --sudo flag and no risk issues
			fmt.Println()
			fmt.Print(dimStyle.Render("  "))
			fmt.Print(tagStyle.Render(" sudo "))
			fmt.Print(" ")
			fmt.Print(cyanStyle.Render("❯"))
			fmt.Print(" ")
			fmt.Println(whiteStyle.Render(trimmed))
		
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
			fmt.Print(dimStyle.Render("  "))
			fmt.Print(cyanStyle.Render("❯"))
			fmt.Print(" ")
			fmt.Println(whiteStyle.Render(trimmed))
		}

	// Execute the command
	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Prefix = dimStyle.Render("  ◆ ")
	s.Start()
	startTime := time.Now()


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
	duration := time.Since(startTime)
	fmt.Print("\r\033[K") // Clear the spinner line

	fmt.Println()

	if err != nil {
		return fmt.Errorf("command execution failed: %w", err)
	}

	fmt.Print(successStyle.Render("  ✓ SUCCESS"))
	fmt.Print(" ")
	fmt.Printf("%s\n", dimStyle.Render(fmt.Sprintf("• executed in %.1fs", duration.Seconds())))
	fmt.Println()
	

	return nil
}
