package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var outputCh = make(chan string)

func main() {

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

	output := <-outputCh
	fmt.Println(output)
}

type model struct {
	Quitting bool
	text     string
	status   int
	output   string
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

	return model{
		Quitting: false,
		text:     "",
		status:   0,
		output:   "",
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {

		case "ctrl+c":
			return m, tea.Quit

		case "enter":
			m.Quitting = true

			return m, runOnTerminal("brew update")
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

	}
	return m, nil
}

func (m model) View() string {

	s := ""

	if m.output != "" {
		return fmt.Sprintf("%s\n\n%s", s, m.output)
	}

	if m.Quitting {
		s := "wait..."
		return s
	}

	if m.output != "" {
		return fmt.Sprintf("%s\n\n%s", s, m.output)
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

func runOnTerminal(command string) tea.Cmd {
	return func() tea.Msg {
		parts := strings.Fields(command)
		c := exec.Command(parts[0], parts[1:]...) //nolint:gosec
		output, err := c.Output()
		if err != nil {
			// fmt.Print(err)
			return CommandMsg{err: err}
		}

		return CommandMsg{
			output: string(output),
		}

	}
}

func sendOutput(output string) tea.Cmd {
	return func() tea.Msg {
		outputCh <- output
		return nil
	}
}
