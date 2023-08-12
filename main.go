package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/sashabaranov/go-openai"
)

func main() {

	configsFlag := flag.Bool("configs", false, "User configs of the application")
	clearStoreFlag := flag.Bool("clear-store", false, "Clear the history store")
	// openStoreFileFlag := flag.Bool("open-store-file", false, "Open the history store file in the default editor")
	flag.Parse()

	if *configsFlag {

		is_open_ai_key_set := ""
		if len(os.Getenv("OPENAI_API_KEY")) > 0 {
			is_open_ai_key_set = "✅"
		} else {
			is_open_ai_key_set = "❌"
		}

		renderer, _ := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
		)

		content := `

# Configs

---

**App config directory**: ` + "`" + getAppConfigDir() + "`" + `
	
---
**OpenAI API key is set?**: ` + is_open_ai_key_set + `


`

		str, _ := renderer.Render(content)

		fmt.Println(str)
		os.Exit(0)

	}

	if *clearStoreFlag {

		fmt.Println("Are you sure you want to clear the history store? [y/N]")
		var response string
		fmt.Scan(&response)
		if response != "y" {
			fmt.Println("Aborting")
			os.Exit(0)
		}

		err := os.Remove(filepath.Join(getAppConfigDir(), store_file_location))
		if err != nil {
			fmt.Printf("Error clearing history file: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ History store cleared")
		os.Exit(0)
	}

	// if *openStoreFileFlag {
	// 	file_path := filepath.Join(getAppConfigDir(), store_file_location)
	// 	println(file_path)
	// 	// err := exec.Command("open", "/Users/mr_senor/Library/Application Support/clAI/store.json").Run()
	// 	err := exec.Command("open", file_path).Run()
	// 	if err != nil {
	// 		fmt.Printf("Error opening history file: %v\n", err)
	// 		os.Exit(1)
	// 	}
	// 	os.Exit(0)
	// }

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
	loading_spinner                   spinner.Model
	loading_timer                     time.Time
	loading_duration                  float64
	prompt_textarea                   textarea.Model
	prompt_screen_err                 string
	is_making_gpt_code_request        bool
	selected_screen                   string
	prompt_response_screen_err        string
	response_code_text                string // chatGPT response to the prompt as markdown code
	response_code_viewport            viewport.Model
	response_code_textInput           textinput.Model
	running_command_screen_err        string
	help                              help.Model
	command_explanation_text          string
	is_making_gpt_explanation_request bool
	explanation_result_viewport       viewport.Model
	history_list                      list.Model
	terminal_width                    int
	terminal_height                   int
}

const store_file_location = "store.json"

// for json umarshall(decode) to work we need to have the fields exported
// ie, start with a capital letter and also need to tell which fields to use
// in the json with the `json:"field_name"` syntax
type history_list_item struct {
	CreatedAt           time.Time `json:"created_at"`
	PromptText          string    `json:"prompt_text"`
	ResponseCode        string    `json:"response_code"`
	ResponseExplanation string    `json:"response_explanation"`
}

func (i history_list_item) Title() string       { return i.PromptText }
func (i history_list_item) Description() string { return "" }
func (i history_list_item) FilterValue() string { return i.PromptText }

var history_list_style = lipgloss.NewStyle().Margin(1, 2)
var screen_style = lipgloss.NewStyle().Margin(1, 2)

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

	code_blocks_border_color := "33"

	explanation_result_viewport := viewport.New(78, 10)
	explanation_result_viewport.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(code_blocks_border_color))

	response_code_viewport := viewport.New(78, 7)
	response_code_viewport.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(code_blocks_border_color))

	loading_spinner := spinner.New()
	loading_spinner.Spinner = spinner.Moon

	history_list := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	history_list.Title = "Your past queries"

	return model{
		loading_spinner:                   loading_spinner,
		prompt_textarea:                   prompt_textarea,
		prompt_screen_err:                 "",
		selected_screen:                   "prompt_screen",
		is_making_gpt_code_request:        false,
		prompt_response_screen_err:        "",
		response_code_text:                "",
		response_code_textInput:           response_code_textInput,
		response_code_viewport:            response_code_viewport,
		command_explanation_text:          "",
		explanation_result_viewport:       explanation_result_viewport,
		is_making_gpt_explanation_request: false,
		running_command_screen_err:        "",
		history_list:                      history_list,
		help:                              help.New(),
	}
}

func (m model) Init() tea.Cmd {
	// I/O we want to perform right as the program is starting
	return tea.Batch(
		textarea.Blink,
		m.loading_spinner.Tick,
		initAppConfigDir,
	)
}

/**
 * "Game loop" that processes user input, events, etc and updates the modal and runs commands
 */
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			// make sure the channel is not blocking after we exit
			return m, tea.Sequence(tea.Quit, sendOutputToChannel("Bye!"))
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.loading_spinner, cmd = m.loading_spinner.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.terminal_width = msg.Width
		m.terminal_height = msg.Height

	}

	// offload lefover update to the selected screen
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

				if m.prompt_textarea.Value() == "" {
					m.prompt_screen_err = "❌ Prompt cannot be empty"
					return m, nil
				}

				m.loading_timer = time.Now()
				m.is_making_gpt_code_request = true

				return m, makeGPTcommandRequest(m.prompt_textarea.Value())

			case "ctrl+h":

				m.selected_screen = "history_screen"
				return m, loadHistoryFromFile

			default:
				if !m.prompt_textarea.Focused() {
					cmd = m.prompt_textarea.Focus()
					cmds = append(cmds, cmd)
				}
				m.prompt_screen_err = ""
			}

		case GPTcommandResult:

			m.loading_duration = time.Since(m.loading_timer).Seconds()

			m.response_code_text = msg.content

			m.response_code_viewport.SetContent(renderResponseCodeViewport(m.response_code_text))

			m.selected_screen = "prompt_response_screen"

			m.is_making_gpt_code_request = false

			return m, appendToHistory(
				history_list_item{
					PromptText:   m.prompt_textarea.Value(),
					ResponseCode: m.response_code_text,
				},
			)

		case GPTcommandError:
			m.loading_duration = time.Since(m.loading_timer).Seconds()
			m.is_making_gpt_code_request = false
			m.prompt_screen_err = "❌ " + msg.err.Error()
			return m, nil
		}

		m.prompt_textarea, cmd = m.prompt_textarea.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case "prompt_response_screen":
		var cmd tea.Cmd

		switch msg := msg.(type) {

		case tea.KeyMsg:
			switch msg.String() {

			case "enter":
				m.loading_timer = time.Now()
				m.selected_screen = "running_command_screen"

				return m, runOnTerminal(m.response_code_text)

			case "e":

				m.loading_timer = time.Now()
				m.is_making_gpt_explanation_request = true

				return m, makeGPTexplanationRequest(m.response_code_text)

			case "esc":
				m.prompt_textarea.Focus()
				m.response_code_text = ""
				m.command_explanation_text = ""
				m.prompt_response_screen_err = ""
				m.selected_screen = "prompt_screen"
				return m, textarea.Blink

			case "m":
				m.response_code_textInput.SetValue(m.response_code_text)
				m.selected_screen = "response_edit_screen"
				return m, nil

			case "c":
				return m, copyCommandToClipboard(m.response_code_text)

			default:
				m.explanation_result_viewport, cmd = m.explanation_result_viewport.Update(msg)
				return m, cmd

			}

		case GPTexplanationResult:

			m.loading_duration = time.Since(m.loading_timer).Seconds()

			m.command_explanation_text = msg.content

			m.explanation_result_viewport.SetContent(renderExplanationResultViewport(m.command_explanation_text))

			m.is_making_gpt_explanation_request = false

			return m, storeExplanationInHistory(m.command_explanation_text)

		case GPTexplanationError:
			m.loading_duration = time.Since(m.loading_timer).Seconds()
			m.is_making_gpt_explanation_request = false
			m.prompt_response_screen_err = "❌ " + msg.err.Error()

		case copyCommandToClipboardResult:
			return m, tea.Sequence(tea.Quit, sendOutputToChannel(msg.output))

		case copyCommandToClipboardError:
			m.prompt_response_screen_err = msg.err.Error()
		}

	case "running_command_screen":
		switch msg := msg.(type) {

		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				m.selected_screen = "prompt_response_screen"
				m.running_command_screen_err = ""
				return m, nil
			}

		case RuOnTerminalResultMsg:
			m.loading_duration = time.Since(m.loading_timer).Seconds()
			outputMsg := "\n\n" + m.response_code_text + fmt.Sprintf("\n\nTook %.1fs\n\n", m.loading_duration) + msg.output
			return m, tea.Sequence(tea.Quit, sendOutputToChannel(outputMsg))

		case RuOnTerminalErrorMsg:
			m.loading_duration = time.Since(m.loading_timer).Seconds()
			m.running_command_screen_err = "❌ " + msg.err.Error()
			return m, nil

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

	case "history_screen":
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c":
				m.selected_screen = "prompt_screen"
				return m, nil

			case "enter":
				selected := m.history_list.SelectedItem().(history_list_item)

				m.response_code_text = selected.ResponseCode
				m.command_explanation_text = selected.ResponseExplanation

				m.response_code_viewport.SetContent(renderResponseCodeViewport(m.response_code_text))

				m.explanation_result_viewport.SetContent(renderExplanationResultViewport(m.command_explanation_text))

				m.selected_screen = "prompt_response_screen"
				return m, nil
			}
		case tea.WindowSizeMsg:
			w, h := history_list_style.GetFrameSize()
			m.history_list.SetSize(msg.Width-w, msg.Height-h)
			return m, nil

		case HistoryFromFileResult:

			items := make([]list.Item, len(msg.history))

			for i, item := range msg.history {
				items[len(items)-1-i] = history_list_item{
					PromptText:          item.PromptText,
					ResponseCode:        item.ResponseCode,
					ResponseExplanation: item.ResponseExplanation,
				}
			}
			m.history_list.SetItems(items)

			w, h := history_list_style.GetFrameSize()
			m.history_list.SetSize(m.terminal_width-w, m.terminal_height-h)

			return m, nil
		}

		var cmd tea.Cmd
		m.history_list, cmd = m.history_list.Update(msg)
		return m, cmd
	}

	return m, nil
}

/**
* "Game loop" that updates the screen everytime the model changes.
*  The function returns a string of the UI to be rendered
 */
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
			s += m.loading_spinner.View() + " Making request..." + fmt.Sprintf(" %.1fs\n\n", time.Since(m.loading_timer).Seconds())
		}

		// The footer
		s += strings.Repeat("\n", 4)
		s += m.help.FullHelpView([][]key.Binding{
			{
				key.NewBinding(
					key.WithKeys("ctrl+s"),
					key.WithHelp("[ ctrl+s ]", "✔︎ Start"),
				),
				key.NewBinding(
					key.WithKeys("ctrl+h"),
					key.WithHelp("[ ctrl+h ]", "⍞ History"),
				),

				key.NewBinding(
					key.WithKeys("ctrl+c"),
					key.WithHelp("[ ctrl+c ]", "⏏︎ Exit"),
				),
			},
		})

		// Send the UI for rendering
		return screen_style.Render(s)

	case "prompt_response_screen":
		s := "Result\n"

		s += m.response_code_viewport.View()

		if m.prompt_response_screen_err != "" {
			s += "\n\n"
			s += m.prompt_response_screen_err
		}

		if m.is_making_gpt_explanation_request {
			s += "\n\n"
			s += m.loading_spinner.View() + " Loading explanation..." + fmt.Sprintf(" %.1fs\n\n", time.Since(m.loading_timer).Seconds())
		}

		if m.command_explanation_text != "" && !m.is_making_gpt_explanation_request {
			s += "\n\n"
			s += "Explanation\n"
			s += m.explanation_result_viewport.View()
		}

		if !m.is_making_gpt_code_request && !m.is_making_gpt_explanation_request {
			s += fmt.Sprintf("\n\nTook %.1fs\n\n", m.loading_duration)
		}

		// The footer
		s += strings.Repeat("\n", 4)

		s += m.help.FullHelpView([][]key.Binding{
			{

				key.NewBinding(
					key.WithKeys("enter"),
					key.WithHelp("[  enter  ]", "✔︎ Run"),
				),
				key.NewBinding(
					key.WithKeys("e"),
					key.WithHelp("[  e      ]", "␦ Explain code"),
				),
				key.NewBinding(
					key.WithKeys("m"),
					key.WithHelp("[  m      ]", "✎ Modify code"),
				),

				key.NewBinding(
					key.WithKeys("c"),
					key.WithHelp("[  c      ]", "☑︎ Copy code to clipboard"),
				),
				key.NewBinding(
					key.WithKeys("esc"),
					key.WithHelp("[  esc    ]", "↩︎ Go back and amend prompt"),
				),
				key.NewBinding(
					key.WithKeys("ctrl+c"),
					key.WithHelp("[  ctrl+c ]", "⏏︎ Exit"),
				),
			},
		})
		return screen_style.Render(s)

	case "running_command_screen":
		s := "Running command: " + m.response_code_text + "\n\n"

		if m.running_command_screen_err != "" {
			s += "\n\n"
			s += m.running_command_screen_err

			s += fmt.Sprintf("\n\nTook %.1fs\n\n", m.loading_duration)

			// The footer
			s += strings.Repeat("\n", 4)
			s += m.help.FullHelpView([][]key.Binding{
				{
					key.NewBinding(
						key.WithKeys("esc"),
						key.WithHelp("[ esc ]", "↩︎ Go back"),
					),
					key.NewBinding(
						key.WithKeys("ctrl+c"),
						key.WithHelp("[ ctrl+c ]", "⏏︎ Exit"),
					),
				},
			})

		} else {
			s += m.loading_spinner.View() + " Processing..." + fmt.Sprintf(" %.1fs\n\n", time.Since(m.loading_timer).Seconds())
		}
		return screen_style.Render(s)

	case "response_edit_screen":
		s := "Edit the result command\n\n"

		s += m.response_code_textInput.View()

		// The footer
		s += strings.Repeat("\n", 4)
		s += m.help.FullHelpView([][]key.Binding{
			{
				key.NewBinding(
					key.WithKeys("enter"),
					key.WithHelp("[ enter  ]", "✔︎ Save"),
				),
				key.NewBinding(
					key.WithKeys("esc"),
					key.WithHelp("[ esc    ]", "↩︎ Go back"),
				),
				key.NewBinding(
					key.WithKeys("ctrl+c"),
					key.WithHelp("[ ctrl+c ]", "⏏︎ Exit"),
				),
			},
		})
		return screen_style.Render(s)

	case "history_screen":
		return history_list_style.Render(m.history_list.View())

	default:
		return ""

	}

}

type RuOnTerminalResultMsg struct {
	output string
}

type RuOnTerminalErrorMsg struct {
	err error
}

func runOnTerminal(command string) tea.Cmd {
	return func() tea.Msg {

		c := exec.Command("bash", "-c", command)

		var stdout, stderr bytes.Buffer
		c.Stdout = &stdout
		c.Stderr = &stderr

		err := c.Run()

		// debug
		// time.Sleep(1 * time.Second)

		if err != nil {
			return RuOnTerminalErrorMsg{err: fmt.Errorf(stderr.String())}
		}

		return RuOnTerminalResultMsg{
			output: stdout.String(),
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

		client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

		request_content := `
			You are a helpful command-line interpreter. You receive natural language queries
			and you return the correspondent bash command. And only the command.
			DO NOT RETURN ANY EXPLANATION OR INSTRUCTION. ONLY RETURN THE COMMAND!
			You have access to some information about the system you are returning the
			command for.
			===
			OS: {{.OS}}
			ARCH: {{.ARCH}}
			CURRENT_DATE: {{.CURRENT_DATE}}
			===
			Example:
			USER: how to list files?
			ASSISTANT:
			ls - la
		`

		var buf bytes.Buffer
		t := template.Must(template.New(request_content).Parse(request_content))

		err := t.Execute(&buf, map[string]string{
			"OS":           runtime.GOOS,
			"ARCH":         runtime.GOARCH,
			"CURRENT_DATE": time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		})

		if err != nil {
			return GPTcommandError{err: err}
		}

		result := buf.String()
		result = strings.ReplaceAll(result, "	", "") // remove tabs

		req := openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: result,
				},
			},
		}

		req.Messages = append(req.Messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		})
		resp, err := client.CreateChatCompletion(context.Background(), req)
		if err != nil {
			return GPTcommandError{err: err}

		}

		return GPTcommandResult{
			content: resp.Choices[0].Message.Content,
		}

		//// debug
		// time.Sleep(1 * time.Second)
		// return GPTcommandError{err: fmt.Errorf("potato is not hot!")}
		// return GPTcommandResult{
		// 	content: `ffmpeg -i input.mp4 -vf "select='not(mod(n\,3))'" output.mp4`,
		// }
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
		client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

		req := openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleSystem,
					Content: `
						You are a helpful command-line interpreter. You receive a bash command and
						you return an explanation for it. And only the explanation.
						Keep the answers simple, concise and short.
						Explain the different parts of the command in a markdown list, each item is a different piece of the command or argument.
					`,
				},
			},
		}

		req.Messages = append(req.Messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: code,
		})
		resp, err := client.CreateChatCompletion(context.Background(), req)
		if err != nil {
			return GPTexplanationError{err: err}

		}

		return GPTexplanationResult{
			content: "`" + resp.Choices[0].Message.Content + "`",
		}

		//// debug
		// time.Sleep(1 * time.Second)

		// return GPTexplanationError{err: fmt.Errorf("potato is not so hot!")}

		// 		return GPTexplanationResult{
		// 			content: `
		// - ffmpeg: the command
		// - -i: input file
		// - input.mp4: the input file
		// - -vf: video filter
		// - "select='not(mod(n\,3))'": select every third frame out of a lot of frames and you know? I like frames
		// - output.mp4: the output file
		// 			`,
		// 		}

	}
}

type copyCommandToClipboardResult struct {
	output string
}

type copyCommandToClipboardError struct {
	err error
}

func copyCommandToClipboard(command string) tea.Cmd {
	return func() tea.Msg {

		copy_this := "echo %s | pbcopy"

		command_to_run := fmt.Sprintf(copy_this, strconv.Quote(command))

		c := exec.Command("bash", "-c", command_to_run)

		var stdout, stderr bytes.Buffer
		c.Stdout = &stdout
		c.Stderr = &stderr

		err := c.Run()

		if err != nil {
			return copyCommandToClipboardError{err: fmt.Errorf("❌ error copying to clipboard: %w", stderr.String())}
		}
		return copyCommandToClipboardResult{
			output: "✅ Command copied to clipboard!",
		}
	}
}

func renderResponseCodeViewport(code string) string {
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(78),
	)

	str, _ := renderer.Render(fmt.Sprintf("```bash\n%s\n```", code))

	return str
}

func renderExplanationResultViewport(explanation string) string {
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		// glamour.WithWordWrap(60),
	)

	str, _ := renderer.Render(explanation)

	return str
}

type HistoryFromFileResult struct {
	history []history_list_item
}

/**
* Loads the store.json into a history_list_item struct
**/
func LoadStore() []history_list_item {
	var historyList []history_list_item

	// load json file
	file, err := os.Open(filepath.Join(getAppConfigDir(), store_file_location))
	if err != nil {
		// return empty_result
		fmt.Printf("Error loading history file: %v\n", err)
		return []history_list_item{}
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&historyList)
	if err != nil {
		fmt.Printf("Error decoding JSON: %v\n", err)
		return []history_list_item{}
	}
	return historyList
}

/**
* Saves an array of history_list_item struct into store.json
**/
func SaveStore(historyList []history_list_item) {

	// save json file
	file, err := os.Create(filepath.Join(getAppConfigDir(), store_file_location))
	if err != nil {
		fmt.Printf("Error creating history file: %v\n", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(historyList)
	if err != nil {
		fmt.Printf("Error encoding JSON: %v\n", err)
		return
	}
}

func loadHistoryFromFile() tea.Msg {

	historyList := LoadStore()

	if historyList == nil {
		return HistoryFromFileResult{
			history: []history_list_item{},
		}
	}

	return HistoryFromFileResult{
		history: historyList,
	}
}

func appendToHistory(item history_list_item) tea.Cmd {
	return func() tea.Msg {
		item.CreatedAt = time.Now()

		historyList := LoadStore()

		if historyList != nil {
			historyList = append(historyList, item)

			SaveStore(historyList)
		}

		return nil

	}
}

func storeExplanationInHistory(explanation string) tea.Cmd {
	return func() tea.Msg {
		historyList := LoadStore()

		if historyList != nil {
			lastItem := historyList[len(historyList)-1]

			lastItem.ResponseExplanation = explanation

			historyList[len(historyList)-1] = lastItem

			SaveStore(historyList)
		}

		return nil

	}
}

func getAppConfigDir() string {
	appConfigDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Printf("Error getting user config dir: %v\n", err)
		return ""
	}

	appConfigDir = filepath.Join(appConfigDir, "clAI")

	return appConfigDir
}

func initAppConfigDir() tea.Msg {

	err := os.MkdirAll(getAppConfigDir(), 0755)
	if err != nil {
		fmt.Printf("Error creating clai directory: %v\n", err)
		return nil
	}

	return nil

}
