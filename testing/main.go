package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/sashabaranov/go-openai"
)

var outputCh = make(chan string)

func main() {

	// command := `find . -type f -access +182 -print`
	// command := strconv.Quote(`ffmpeg -i input.mp4 -vf "select='not(mod(n,4))',setpts=N/FRAME_RATE/TB" output.mp4`)

	// copy_this := "echo '%s' | pbcopy"

	// command_to_run := fmt.Sprintf(copy_this, command)

	// fmt.Printf("command to run: %s", command_to_run)

	// c := exec.Command("bash", "-c", command_to_run)

	// var stdout, stderr bytes.Buffer
	// c.Stdout = &stdout
	// c.Stderr = &stderr

	// err := c.Run()
	// if err != nil {
	// 	fmt.Print(err)
	// 	fmt.Print(stderr.String())
	// }

	// fmt.Println("\n\n\nstdout:", stdout.String())
	// os.Exit(0)

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

	// output := <-outputCh
	// fmt.Println(output)
}

type model struct {
	Quitting bool
	text     string
	status   int
	output   string
	spinner  spinner.Model
	viewport viewport.Model
	list     list.Model
}

type httpMsg struct {
	status int
	err    error
	text   string
}

type CommandMsg struct {
	err    error
	output string
}

type Item struct {
	prompt     string
	created_at string
}

var docStyle = lipgloss.NewStyle().Margin(1, 2)

func (i Item) Title() string       { return i.created_at }
func (i Item) Description() string { return i.prompt }
func (i Item) FilterValue() string { return i.prompt }

func initialModel() model {

	vp := viewport.New(100, 7)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		// glamour.WithWordWrap(20),
	)

	// str, _ := renderer.Render(fmt.Sprintf("```bash\n%s\n```", `ls -la`))
	str, _ := renderer.Render(fmt.Sprintf("```console\n%s\n```", `ffmpeg -i input.mp4 -vf "select='not(mod(n\,3))'" output.mp4`))
	// str, _ := renderer.Render(fmt.Sprintf("```python\n%s\n```", `import os; os.system("ffmpeg -i input.mp4 -vf 'select=\\'not(mod(n\\,3))\\'' output.mp4")`))
	vp.SetContent(str)

	s := spinner.New()
	s.Spinner = spinner.Moon

	items := []list.Item{
		Item{prompt: "How to get the first frame of a video?", created_at: "2021-04-12 12:00"},
		Item{prompt: "How to get the last frame of a video?", created_at: "2021-04-12 12:00"},
		Item{prompt: "How to get every 3rd frame of a video?", created_at: "2021-04-12 12:00"},
		Item{prompt: "How to get every 5rd frame of a video?", created_at: "2021-04-12 12:00"},
		Item{prompt: "Potatoes are great mate! you know potatows are nice because potatoes!", created_at: "2021-04-12 12:00"},
	}

	that_list := list.New(items, list.NewDefaultDelegate(), 0, 0)
	that_list.Title = "A big list of nothing!"

	return model{
		Quitting: false,
		text:     "",
		status:   0,
		viewport: vp,
		output:   "",
		spinner:  s,
		list:     that_list,
	}
}

func (m model) Init() tea.Cmd {
	// return m.spinner.Tick
	return loadHistoryFromFile
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {

		case "ctrl+c":
			return m, tea.Quit

		case "enter":
			return m, loadHistoryFromFile
			// i, ok := m.list.SelectedItem().(Item)
			// if ok {

			// 	fmt.Println(i.prompt)
			// 	return m, nil
			// }

		}

	case httpMsg:
		m.status = msg.status
		m.text = msg.text
		return m, nil

	case HistoryFromFileResult:
		fmt.Printf("%v\n", msg.history)
		m.output = fmt.Sprintf("%v\n", msg.history)
		return m, nil

	case CommandMsg:
		m.output = msg.output
		return m, tea.Sequence(tea.Quit, sendOutput(string(msg.output)))
		// return m, tea.Sequence(tea.Quit, sendOutput(string(msg.output)))

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

		// default:
		// var cmd tea.Cmd
		// m.spinner, cmd = m.spinner.Update(msg)
		// return m, cmd

	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
	// return m, nil
}

func (m model) View() string {

	// return docStyle.Render(m.list.View())

	s := ""

	// s += m.spinner.View()

	s += "\n\n"

	s += m.viewport.View()

	if m.output != "" {
		return m.output
	}

	if m.Quitting {
		s := "wait..."
		return s
	}

	// footer
	s += "\n\nPress enter to do whatever."
	s += "\n\nPress ctrl+c to quit."
	return s

}

type history_list_item struct {
	CreatedAt           time.Time `json:"created_at"`
	PromptText          string    `json:"prompt_text"`
	ResponseCode        string    `json:"response_code"`
	ResponseExplanation string    `json:"response_explanation"`
}

type HistoryFromFileResult struct {
	history []history_list_item
}

func loadHistoryFromFile() tea.Msg {

	var historyList []history_list_item

	// empty_result := HistoryFromFileResult{
	// 	history: []history_list_item{},
	// }

	// load json file
	file, err := os.Open(".clai/store.json")
	if err != nil {
		// return empty_result
		fmt.Printf("Error loading history file: %v\n", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&historyList)
	if err != nil {
		fmt.Printf("Error decoding JSON: %v\n", err)
	}

	return HistoryFromFileResult{
		history: historyList,
	}
}

func makeRequest(url string) tea.Cmd {
	return func() tea.Msg {
		c := &http.Client{
			Timeout: 10 * time.Second,
		}
		res, err := c.Get(url)
		if err != nil {
			fmt.Print(err)
		}
		defer res.Body.Close()

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Print(err)
		}
		// fmt.Println(string(body))

		return httpMsg{
			status: res.StatusCode,
			err:    err,
			text:   string(body),
		}

	}
}

type GPTRequest string

func makeGPTcommandRequest() tea.Cmd {
	return func() tea.Msg {

		client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

		req := openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleSystem,
					Content: `
						You are a helpful command-line interpreter. You receive natural language queries
						and you return the correspondent bash command. And only the command.
						You have access to some information about the system you are returning the 
						command for.
						===
						OS: darwin
						ARCH: aarch64
						CURRENT_DATE: 2023-07-21T20:43:53Z
						===

						Example:
						USER: how to list files?

						ASSISTANT:
						ls - la`,
				},
			},
		}

		req.Messages = append(req.Messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: "how to get every third frame of a mp4 video?",
		})
		resp, err := client.CreateChatCompletion(context.Background(), req)
		if err != nil {
			fmt.Printf("ChatCompletion error: %v\n", err)
		}

		fmt.Printf("%s\n\n", resp.Choices[0].Message.Content)

		return GPTRequest("ok")
	}
}

func runOnTerminal(command string) tea.Cmd {
	return func() tea.Msg {
		// command = "find . -name *.go | xargs wc -l"
		// command = `find . -type f -print0 | xargs -0 stat -f "%m %N" | sort -rn | head -n 10 | cut -d" " -f2-`
		// command = `find . -name "*.gif" -type f -exec stat -f "%Sm %N" {} \; | sort -nr | awk '{print $2}'`
		command = `find . -type f -access +182 -print`

		// command = strings.ReplaceAll(command, "\"", "")

		// parts := strings.Fields(command)
		// c := exec.Command(parts[0], parts[1:]...) //nolint:gosec
		c := exec.Command("bash", "-c", command) //nolint:gosec

		var stdout, stderr bytes.Buffer
		c.Stdout = &stdout
		c.Stderr = &stderr

		err := c.Run()

		fmt.Print("Err: ", err)
		fmt.Println("\n\nstdout:", stdout.String())
		fmt.Println("\n\nstderr:", stderr.String())

		// c := exec.Command("find", ".", "-name", "*.go")
		// c := exec.Command("ls", "-la")
		// c := exec.Command("pwd")
		// output, err := c.Output()
		// if err != nil {
		// 	fmt.Print(err)
		// 	// return CommandMsg{err: err}
		// }

		// fmt.Println(string(output))

		return nil
		// return CommandMsg{
		// 	output: string(output),
		// }

	}
}

func sendOutput(output string) tea.Cmd {
	return func() tea.Msg {
		outputCh <- output
		return nil
	}
}
