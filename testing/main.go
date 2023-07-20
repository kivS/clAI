package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
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

type keymap struct {
	start   key.Binding
	run     key.Binding
	modify  key.Binding
	explain key.Binding
	copy    key.Binding
	go_back key.Binding
	exit    key.Binding
}

type model struct {
	textarea        textarea.Model
	err             error
	prompting       bool
	selected_screen string
	help            help.Model
	keymap          keymap
}

func initialModel() model {
	ti := textarea.New()
	ti.Placeholder = "Once upon a time..."
	ti.Focus()

	return model{
		textarea:        ti,
		err:             nil,
		selected_screen: "prompt_screen",
		help:            help.New(),
		keymap: keymap{
			start: key.NewBinding(
				key.WithKeys("ctrl+s"),
				key.WithHelp("ctrl+s", "Start"),
			),
			run: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "Run on commandline"),
			),
			modify: key.NewBinding(
				key.WithKeys("m"),
				key.WithHelp("m", "Modify command"),
			),
			explain: key.NewBinding(
				key.WithKeys("e"),
				key.WithHelp("e", "Explain command"),
			),
			copy: key.NewBinding(
				key.WithKeys("c"),
				key.WithHelp("c", "Copy command to clipboard"),
			),
			go_back: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "Go back"),
			),
			exit: key.NewBinding(
				key.WithKeys("ctrl+c"),
				key.WithHelp("ctrl+c", "Exit"),
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
			switch {
			case key.Matches(msg, m.keymap.exit):
				return m, tea.Quit
			case key.Matches(msg, m.keymap.go_back):
				m.textarea.Focus()
				m.textarea.SetValue("")
				m.selected_screen = "prompt_screen"
				return m, nil

			}
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

		s += "> rmlol -rf /"

		// The footer
		// s += "\n\n"
		// s += "\n(enter to run code) / (e to explain code) / (r to redo prompt) / (ctrl+c to quit) \n"
		s += "\n\n"

		s += m.help.FullHelpView([][]key.Binding{
			{
				m.keymap.go_back,
				m.keymap.exit,
			},
		})
		return s

	default:
		return ""

	}

}
