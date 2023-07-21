package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

type errMsg error

type help_keymap struct {
	start   key.Binding
	save    key.Binding
	run     key.Binding
	modify  key.Binding
	explain key.Binding
	copy    key.Binding
	go_back key.Binding
	exit    key.Binding
}

type model struct {
	textarea                    textarea.Model
	prompt_screen_err           string
	prompting                   bool
	selected_screen             string
	response_code_text          string // response to the prompt as code
	response_code_viewport      viewport.Model
	response_code_textInput     textinput.Model
	help                        help.Model
	help_keymap                 help_keymap
	command_explanation_text    string
	explanation_result_viewport viewport.Model
}

func initialModel() model {
	ti := textarea.New()
	ti.ShowLineNumbers = false
	ti.Placeholder = "How to..."

	ti.Focus()

	ti2 := textinput.New()
	ti2.Focus()
	ti2.CharLimit = 156
	ti2.Width = 0

	vp := viewport.New(78, 20)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		PaddingRight(2)

	vp2 := viewport.New(100, 5)
	vp2.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	return model{
		textarea:                    ti,
		prompt_screen_err:           "",
		selected_screen:             "prompt_screen",
		response_code_text:          `say "hello potato"`,
		response_code_textInput:     ti2,
		response_code_viewport:      vp2,
		command_explanation_text:    "",
		explanation_result_viewport: vp,
		help:                        help.New(),
		help_keymap: help_keymap{
			start: key.NewBinding(
				key.WithKeys("ctrl+s"),
				key.WithHelp("[ ctrl+s ]", "Start"),
			),
			save: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("[ enter  ]", "Save"),
			),
			run: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("[ enter  ]", "Run on commandline"),
			),
			modify: key.NewBinding(
				key.WithKeys("m"),
				key.WithHelp("[ m      ]", "Modify command"),
			),
			explain: key.NewBinding(
				key.WithKeys("e"),
				key.WithHelp("[ e      ]", "Explain command"),
			),
			copy: key.NewBinding(
				key.WithKeys("c"),
				key.WithHelp("[ c      ]", "Copy command to clipboard"),
			),
			go_back: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("[ esc    ]", "Go back"),
			),
			exit: key.NewBinding(
				key.WithKeys("ctrl+c"),
				key.WithHelp("[ ctrl+c ]", "Exit"),
			),
		},
	}
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(msg, m.help_keymap.exit) {
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
			switch {

			case key.Matches(msg, m.help_keymap.start):

				if m.textarea.Value() == "" {
					m.prompt_screen_err = "❌ Prompt cannot be empty"
					return m, nil
				}

				renderer, _ := glamour.NewTermRenderer(
					glamour.WithAutoStyle(),
					// glamour.WithWordWrap(20),
				)

				str, _ := renderer.Render(m.response_code_text)
				m.response_code_viewport.SetContent(str)

				m.selected_screen = "prompt_response_screen"

				return m, nil

			default:
				if !m.textarea.Focused() {
					cmd = m.textarea.Focus()
					cmds = append(cmds, cmd)
				}
				m.prompt_screen_err = ""
			}
		}

		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case "prompt_response_screen":
		var cmd tea.Cmd

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {

			case key.Matches(msg, m.help_keymap.run):
				return m, tea.Quit

			case key.Matches(msg, m.help_keymap.explain):
				m.command_explanation_text = `
# Hello, World!

## This is a test This is a testThis is a testThis is a testThis is a test

[link](https://example.com)

- list item 1
- list item 2
- list item 3
- list item 4
	- list item 1
	- list item 2
	- list item 3
	- list item 4

> This is a blockquote
> This is a blockquote
> This is a blockquote

This is a test This is a testThis is a testThis is a testThis is a test
This is a test This is a testThis is a testThis is a testThis is a test

This is a test This is a testThis is a testThis is a testThis is a test
This is a test This is a testThis is a testThis is a testThis is a test

#### This is a test This is a testThis is a testThis is a testThis is a test
#### This is a test This is a testThis is a testThis is a testThis is a test


				`

				renderer, _ := glamour.NewTermRenderer(
					glamour.WithAutoStyle(),
					glamour.WithWordWrap(78),
				)

				str, _ := renderer.Render(m.command_explanation_text)
				m.explanation_result_viewport.SetContent(str)

				return m, nil

			case key.Matches(msg, m.help_keymap.go_back):
				m.textarea.Focus()
				m.textarea.SetValue("")
				m.selected_screen = "prompt_screen"
				return m, nil

			case key.Matches(msg, m.help_keymap.modify):
				m.response_code_textInput.SetValue(m.response_code_text)
				m.selected_screen = "response_edit_screen"
				return m, nil

			default:
				m.explanation_result_viewport, cmd = m.explanation_result_viewport.Update(msg)
				return m, cmd

			}
		}

	case "response_edit_screen":
		var cmd tea.Cmd

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.help_keymap.go_back):
				m.selected_screen = "prompt_response_screen"
				return m, nil

			case key.Matches(msg, m.help_keymap.save):
				if m.response_code_textInput.Value() != m.response_code_text {
					m.response_code_text = m.response_code_textInput.Value()

					// re-render the code in the viewport
					m.response_code_viewport.SetContent(m.response_code_text)

					// we just updated the code so the explanation is no longer valid
					m.command_explanation_text = ""
				}
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
		s := "Your prompt\n\n"

		s += m.textarea.View()

		if m.prompt_screen_err != "" {
			s += "\n\n"
			s += m.prompt_screen_err
		}

		// The footer
		s += "\n\n"
		s += m.help.FullHelpView([][]key.Binding{
			{
				m.help_keymap.start,
				m.help_keymap.exit,
			},
		})

		// Send the UI for rendering
		return s

	case "prompt_response_screen":
		s := "Result\n\n"

		s += m.response_code_viewport.View()

		if m.command_explanation_text != "" {
			s += "\n\n"
			s += m.explanation_result_viewport.View()
		}

		// The footer
		s += "\n\n"
		s += m.help.FullHelpView([][]key.Binding{
			{
				m.help_keymap.run,
				m.help_keymap.explain,
				m.help_keymap.modify,
				m.help_keymap.copy,
				m.help_keymap.go_back,
				m.help_keymap.exit,
			},
		})
		return s

	case "response_edit_screen":
		s := "Edit the result command\n\n"

		s += m.response_code_textInput.View()

		// The footer
		s += "\n\n"
		s += m.help.FullHelpView([][]key.Binding{
			{
				m.help_keymap.save,
				m.help_keymap.go_back,
				m.help_keymap.exit,
			},
		})
		return s

	default:
		return ""

	}

}
