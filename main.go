package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
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
	textarea                textarea.Model
	err                     error
	prompting               bool
	selected_screen         string
	response_code_text      string // response to the prompt as code
	response_code_textInput textinput.Model
}

func initialModel() model {
	ti := textarea.New()
	ti.ShowLineNumbers = false
	ti.Placeholder = "How to..."
	ti.Focus()

	ti2 := textinput.New()
	ti2.Focus()
	ti2.CharLimit = 156
	ti2.Width = 20

	return model{
		textarea:                ti,
		err:                     nil,
		selected_screen:         "prompt_screen",
		response_code_text:      `say "hello potato"`,
		response_code_textInput: ti2,
	}
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		k := msg.String()
		if k == "ctrl+c" {
			return m, tea.Quit
		}

	}

	// offload update to the selected screen
	return updateSelectedScreen(msg, m)
}

func updateSelectedScreen(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	switch m.selected_screen {
	case "prompt_screen":
		var cmds []tea.Cmd
		var cmd tea.Cmd

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+s":
				m.selected_screen = "prompt_response_screen"
				return m, nil

			default:
				if !m.textarea.Focused() {
					cmd = m.textarea.Focus()
					cmds = append(cmds, cmd)
				}
			}
		}

		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case "prompt_response_screen":
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				return m, tea.Quit
			case "esc":
				m.textarea.Focus()
				m.textarea.SetValue("")
				m.selected_screen = "prompt_screen"
				return m, nil

			case "m":
				m.response_code_textInput.SetValue(m.response_code_text)
				m.selected_screen = "response_edit_screen"
				return m, nil

			}
		}

	case "response_edit_screen":
		var cmd tea.Cmd

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.selected_screen = "prompt_response_screen"
				return m, nil

			case "enter":
				m.response_code_text = m.response_code_textInput.Value()
				m.selected_screen = "prompt_response_screen"
				return m, nil
			}

			m.response_code_textInput, cmd = m.response_code_textInput.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m model) View() string {

	switch m.selected_screen {
	case "prompt_screen":
		// The header
		s := "Your prompt..\n\n"

		s += m.textarea.View()

		if m.prompting {
			s += "\n\n"
			s += "You entered: " + m.textarea.Value() + "\n"
		}

		// The footer
		s += "\n\n"
		s += "\n(ctrl+s to send) / (ctrl+c to quit)\n"

		// Send the UI for rendering
		return s

	case "prompt_response_screen":
		s := "Your prompt response..\n\n"

		s += "> " + m.response_code_text

		// The footer
		s += "\n\n"
		s += "\n(enter to run code) / (e to explain code)  / (m to edit response code) / (esc to go back to prompt) / (ctrl+c to quit) \n"
		return s

	case "response_edit_screen":
		s := "Edit the response code\n\n"

		s += m.response_code_textInput.View()

		// The footer
		s += "\n\n"
		s += "\n(enter to save) / (esc  to go back) / (ctrl+c to quit) \n"
		return s

	default:
		return ""

	}

}
