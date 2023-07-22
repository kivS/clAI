package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/sashabaranov/go-openai"
)

var outputCh = make(chan string)

func main() {

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

	return model{
		Quitting: false,
		text:     "",
		status:   0,
		viewport: vp,
		output:   "",
		spinner:  s,
	}
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {

		case "ctrl+c":
			return m, tea.Quit

		case "enter":
			// m.Quitting = true

			// return m, nil

			// return m, makeGPTcommandRequest()

			return m, runOnTerminal("")
			// return m, makeRequest("https://charm.sh/")
		}

	case httpMsg:
		m.status = msg.status
		m.text = msg.text
		return m, nil

	case CommandMsg:
		m.output = msg.output
		return m, tea.Sequence(tea.Quit, sendOutput(string(msg.output)))
		// return m, tea.Sequence(tea.Quit, sendOutput(string(msg.output)))

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	}
	return m, nil
}

func (m model) View() string {

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
		command = `find . -name "*.gif" -type f -exec stat -f "%Sm %N" {} \; | sort -nr | awk '{print $2}'`

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
