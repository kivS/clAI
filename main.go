package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

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

	// when returning the command we need to output to the user screen
	output := <-outputCh
	fmt.Println(output)
}

type model struct {
	prompt_textarea                   textarea.Model
	prompt_screen_err                 string
	is_making_gpt_code_request        bool
	selected_screen                   string
	response_code_text                string // chatGPT response to the prompt as markdown code
	response_code_viewport            viewport.Model
	response_code_textInput           textinput.Model
	help                              help.Model
	help_keymap                       help_keymap
	command_explanation_text          string
	is_making_gpt_explanation_request bool
	explanation_result_viewport       viewport.Model
}

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

func initialModel() model {
	prompt_textarea := textarea.New()
	prompt_textarea.ShowLineNumbers = false
	prompt_textarea.SetWidth(60)
	prompt_textarea.Placeholder = "How to..."
	prompt_textarea.Focus()

	response_code_textInput := textinput.New()
	response_code_textInput.Focus()
	response_code_textInput.CharLimit = 156
	response_code_textInput.Width = 0

	explanation_result_viewport := viewport.New(78, 10)
	explanation_result_viewport.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		PaddingRight(2)

	response_code_viewport := viewport.New(100, 7)
	response_code_viewport.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	return model{
		prompt_textarea:                   prompt_textarea,
		prompt_screen_err:                 "",
		selected_screen:                   "prompt_screen",
		is_making_gpt_code_request:        false,
		response_code_text:                "",
		response_code_textInput:           response_code_textInput,
		response_code_viewport:            response_code_viewport,
		command_explanation_text:          "",
		explanation_result_viewport:       explanation_result_viewport,
		is_making_gpt_explanation_request: false,
		help:                              help.New(),
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
				key.WithHelp("[ enter  ]", "Run the command"),
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
	// I/O we want to perform right as the program is starting
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(msg, m.help_keymap.exit) {
			// make sure the channel is not blocking after we exit
			return m, tea.Sequence(tea.Quit, sendOutputToChannel("Bye!"))
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

		case GPTcommandResult:

			m.response_code_text = msg.content

			renderer, _ := glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				// glamour.WithWordWrap(20),
			)

			str, _ := renderer.Render(fmt.Sprintf("```bash\n%s\n```", m.response_code_text))

			m.response_code_viewport.SetContent(str)

			m.selected_screen = "prompt_response_screen"

			m.is_making_gpt_code_request = false

			return m, nil

		case tea.KeyMsg:
			switch {

			case key.Matches(msg, m.help_keymap.start):

				if m.prompt_textarea.Value() == "" {
					m.prompt_screen_err = "❌ Prompt cannot be empty"
					return m, nil
				}

				m.is_making_gpt_code_request = true

				return m, makeGPTcommandRequest(m.prompt_textarea.Value())

			default:
				if !m.prompt_textarea.Focused() {
					cmd = m.prompt_textarea.Focus()
					cmds = append(cmds, cmd)
				}
				m.prompt_screen_err = ""
			}
		}

		m.prompt_textarea, cmd = m.prompt_textarea.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case "prompt_response_screen":
		var cmd tea.Cmd

		switch msg := msg.(type) {

		case GPTexplanationResult:

			m.command_explanation_text = msg.content

			renderer, _ := glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				glamour.WithWordWrap(60),
			)

			str, _ := renderer.Render(m.command_explanation_text)
			m.explanation_result_viewport.SetContent(str)

			m.is_making_gpt_explanation_request = false

			return m, nil

		case tea.KeyMsg:
			switch {

			case key.Matches(msg, m.help_keymap.run):
				m.selected_screen = "running_command_screen"
				return m, runOnTerminal(m.response_code_text)

			case key.Matches(msg, m.help_keymap.explain):

				m.is_making_gpt_explanation_request = true

				return m, makeGPTexplanationRequest(m.response_code_text)

			case key.Matches(msg, m.help_keymap.go_back):
				m.prompt_textarea.Focus()
				m.prompt_textarea.SetValue("")
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

	case "running_command_screen":
		switch msg := msg.(type) {
		case CommandResultMsg:

			return m, tea.Sequence(tea.Quit, sendOutputToChannel(msg.output))
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

		s += m.prompt_textarea.View()

		if m.prompt_screen_err != "" {
			s += "\n\n"
			s += m.prompt_screen_err
		}

		if m.is_making_gpt_code_request {
			s += "\n\n"
			s += "wait..."
		}

		// The footer
		s += strings.Repeat("\n", 4)
		s += m.help.FullHelpView([][]key.Binding{
			{
				m.help_keymap.start,
				m.help_keymap.exit,
			},
		})

		// Send the UI for rendering
		return s

	case "prompt_response_screen":
		s := "Result\n"

		s += m.response_code_viewport.View()

		if m.is_making_gpt_explanation_request {
			s += "\n\n"
			s += "Loading explanation..."
		}

		if m.command_explanation_text != "" && !m.is_making_gpt_explanation_request {
			s += "\n\n"
			s += "Explanation\n"
			s += m.explanation_result_viewport.View()
		}

		// The footer
		s += strings.Repeat("\n", 4)
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

	case "running_command_screen":
		s := "Running command: " + m.response_code_text + "\n\n"
		s += "wait...\n\n"
		return s

	case "response_edit_screen":
		s := "Edit the result command\n\n"

		s += m.response_code_textInput.View()

		// The footer
		s += strings.Repeat("\n", 4)
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

type CommandResultMsg struct {
	output string
}

type CommandErrMsg struct {
	err error
}

func runOnTerminal(command string) tea.Cmd {
	return func() tea.Msg {
		parts := strings.Fields(command)
		c := exec.Command(parts[0], parts[1:]...)
		output, err := c.Output()
		if err != nil {
			return CommandErrMsg{err: err}
		}

		return CommandResultMsg{
			output: string(output),
		}

	}
}

var outputCh = make(chan string)

func sendOutputToChannel(output string) tea.Cmd {
	return func() tea.Msg {
		outputCh <- output
		return nil
	}
}

type GPTcommandResult struct {
	content string
}

type GPTcommandError struct {
	err error
}

func makeGPTcommandRequest(prompt string) tea.Cmd {
	return func() tea.Msg {

		// client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

		// req := openai.ChatCompletionRequest{
		// 	Model: openai.GPT3Dot5Turbo,
		// 	Messages: []openai.ChatCompletionMessage{
		// 		{
		// 			Role: openai.ChatMessageRoleSystem,
		// 			Content: `
		// 				You are a helpful command-line interpreter. You receive natural language queries
		// 				and you return the correspondent bash command. And only the command.
		// 				DO NOT RETURN ANY EXPLANATION OR INSTRUCTION. ONLY RETURN THE COMMAND!
		// 				You have access to some information about the system you are returning the
		// 				command for.
		// 				===
		// 				OS: darwin
		// 				ARCH: aarch64
		// 				CURRENT_DATE: 2023-07-21T20:43:53Z
		// 				===

		// 				Example:
		// 				USER: how to list files?

		// 				ASSISTANT:
		// 				ls - la`,
		// 		},
		// 	},
		// }

		// req.Messages = append(req.Messages, openai.ChatCompletionMessage{
		// 	Role:    openai.ChatMessageRoleUser,
		// 	Content: prompt,
		// })
		// resp, err := client.CreateChatCompletion(context.Background(), req)
		// if err != nil {
		// 	return GPTcommandError{err: err}

		// }

		// return GPTcommandResult{
		// 	content: resp.Choices[0].Message.Content,
		// }

		// debug
		time.Sleep(1 * time.Second)
		return GPTcommandResult{
			content: `ffmpeg -i input.mp4 -vf "select='not(mod(n\,3))'" output.mp4`,
		}
	}
}

type GPTexplanationResult struct {
	content string
}

type GPTexplanationError struct {
	err error
}

func makeGPTexplanationRequest(code string) tea.Cmd {
	return func() tea.Msg {
		// client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

		// req := openai.ChatCompletionRequest{
		// 	Model: openai.GPT3Dot5Turbo,
		// 	Messages: []openai.ChatCompletionMessage{
		// 		{
		// 			Role: openai.ChatMessageRoleSystem,
		// 			Content: `
		// 				You are a helpful command-line interpreter. You receive a bash command and
		// 				you return an explanation for it. And only the explanation.
		// 				Keep the answers simple, concise and short.
		// 				Explain the different parts of the command in a markdown list, each item is a different piece of the command or argument.
		// 			`,
		// 		},
		// 	},
		// }

		// req.Messages = append(req.Messages, openai.ChatCompletionMessage{
		// 	Role:    openai.ChatMessageRoleUser,
		// 	Content: code,
		// })
		// resp, err := client.CreateChatCompletion(context.Background(), req)
		// if err != nil {
		// 	return GPTexplanationError{err: err}

		// }

		// return GPTexplanationResult{
		// 	content: resp.Choices[0].Message.Content,
		// }

		// debug
		time.Sleep(1 * time.Second)
		return GPTexplanationResult{
			content: `
- ffmpeg: the command
- -i: input file
- input.mp4: the input file
- -vf: video filter
- "select='not(mod(n\,3))'": select every third frame out of a lot of frames and you know? I like frames
- output.mp4: the output file
			`,
		}

	}
}
