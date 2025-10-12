package executor

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)


type InteractionModel struct {
	textInput       textinput.Model
	Confirmed       bool
	Cancelled       bool
	showSudoWarning bool
	prompt          string
	expectedInput   string
}

func InterationModel(prompt, expectedInput string, showSudoWarning bool) InteractionModel {
	ti := textinput.New()
	ti.Placeholder = expectedInput
	ti.Focus()
	ti.CharLimit = len(expectedInput) + 5
	ti.Width = 50

	return InteractionModel{
		textInput:       ti,
		showSudoWarning: showSudoWarning,
		prompt:          prompt,
		expectedInput:   expectedInput,
	}
}

func (m InteractionModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m InteractionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.Cancelled = true
			return m, tea.Quit
		case "enter":
			input := strings.TrimSpace(strings.ToLower(m.textInput.Value()))
			if m.expectedInput != "" && input == strings.ToLower(m.expectedInput) {
				m.Confirmed = true
			} else if m.expectedInput == "" && (input == "y" || input == "Y") {
				m.Confirmed = true
			} else {
				m.Cancelled = true
			}
			return m, tea.Quit
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m InteractionModel) View() string {
	if m.showSudoWarning {
		return fmt.Sprintf(
			"\n%s\n%s",
			cyanStyle.Render("Proceed with sudo? [y/N]"),
			m.textInput.View(),
		)
	}

	if m.prompt != "" {
		return fmt.Sprintf("%s\n\n%s", m.prompt, m.textInput.View())
	}

	return fmt.Sprintf("%s\n\n", m.textInput.View())
}

