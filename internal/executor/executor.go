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
	commandStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	cancelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	whiteStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	cyanStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	tagStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true).
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
			"\n%s \n\n%s\n%s",
			warningStyle.Render(" ❯ Requires elevated privileges"),
			cyanStyle.Render("Proceed with sudo? [y/N]"),
			m.textInput.View(),
		)
	}

	return fmt.Sprintf(
		"%s\n\n",
		m.textInput.View(),
	)
}

func runSudoAuth() error {
	sudoCmd := exec.Command("sudo", "-v")
	sudoCmd.Stdin = os.Stdin
	sudoCmd.Stdout = os.Stdout
	sudoCmd.Stderr = os.Stderr
	if err := sudoCmd.Run(); err != nil {
		fmt.Println()
		return fmt.Errorf("failed to authenticate with sudo: %w", err)
	}
	return nil
}

func printCommand(cmd string, withSudo bool) {
	fmt.Println()
	fmt.Print(dimStyle.Render("  "))
	if withSudo {
		fmt.Print(tagStyle.Render(" sudo "))
		fmt.Print(" ")
	}
	fmt.Print(cyanStyle.Render("❯"))
	fmt.Print(" ")
	fmt.Println(whiteStyle.Render(cmd))
}

func runCommand(trimmed string) error {
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

func Execute(command string, cfg *config.Config, usedSudoFlag bool) error {
	trimmed := strings.TrimSpace(command)
	assessment := AssessCommandRisk(trimmed, usedSudoFlag)

	needsSudo := strings.HasPrefix(trimmed, "sudo ")
	hasRiskAssessmentIssues := len(assessment.Reasons) > 0

	// Case 1: Risks detected
	if hasRiskAssessmentIssues {
		fmt.Println()
		fmt.Print(warningStyle.Render(" ❯ Command requires caution"))
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
		fmt.Println(cyanStyle.Render("Proceed? [y/N]"))
		// Confirm with Bubble Tea
		p := tea.NewProgram(initialModel(false))
		m, err := p.Run()
		if err != nil {
			return fmt.Errorf("failed to show confirmation prompt: %w", err)
		}
		result := m.(confirmModel)
		if result.cancelled || !result.confirmed {
			//fmt.Println()
			fmt.Print(cancelStyle.Render("  ✗ CANCELLED"))
			fmt.Print(" ")
			fmt.Println(dimStyle.Render("• user aborted"))
			fmt.Println()
			return nil
		}

		// Authenticate sudo if required
		if needsSudo {
			if err := runSudoAuth(); err != nil {
				return err
			}
		}

		printCommand(trimmed, needsSudo)

		// Case 2: Needs sudo but no risk issues
	} else if needsSudo {
		// Show sudo confirmation only if user explicitly passed --sudo
		if usedSudoFlag {
			p := tea.NewProgram(initialModel(true))
			m, err := p.Run()
			if err != nil {
				return fmt.Errorf("failed to show confirmation prompt: %w", err)
			}
			result := m.(confirmModel)
			if result.cancelled || !result.confirmed {
				//fmt.Println()
				fmt.Print(cancelStyle.Render("  ✗ CANCELLED"))
				fmt.Print(" ")
				fmt.Println(dimStyle.Render("• user aborted"))
				fmt.Println()
				return nil
			}
		}

		// Authenticate sudo
		if err := runSudoAuth(); err != nil {
			return err
		}

		printCommand(trimmed, true)

		// Case 3: Normal command
	} else {
		printCommand(trimmed, false)
	}

	// Execute the command
	return runCommand(trimmed)
}
