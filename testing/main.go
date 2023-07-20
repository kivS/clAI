package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

type errMsg error

type model struct {
	textarea  textarea.Model
	err       error
	prompting bool
}

func initialModel() model {
	ti := textarea.New()
	ti.Placeholder = "Once upon a time..."
	ti.Focus()

	return model{
		textarea: ti,
		err:      nil,
	}
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		case "ctrl+s":
			// ctrl+enter will send the message
			m.textarea.Blur()
			m.prompting = true

			return m, nil

		// These keys should exit the program.
		case "ctrl+c":
			return m, tea.Quit

		default:
			if !m.textarea.Focused() {
				cmd = m.textarea.Focus()
				cmds = append(cmds, cmd)
			}

		}
	case errMsg:
		m.err = msg
		return m, nil

	}

	// Return the updated model to the Bubble Tea runtime for processing.
	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	// The header
	s := "Your prompt..\n\n"

	s += m.textarea.View()

	if m.prompting {
		s += "\n\n"
		s += "You entered: " + m.textarea.Value() + "\n"
	}

	// The footer
	s += "\n\n"
	s += "\n(ctrl+c to quit) / (ctrl+s to send)\n"

	// Send the UI for rendering
	return s
}
