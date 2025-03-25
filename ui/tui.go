package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type chatHistory struct {
	messages   []string
	input      []string
	currentPos int
}

func (ch *chatHistory) addMessage(message string) {
	ch.messages = append(ch.messages, message)
}

func (ch *chatHistory) addInput(input string) {
	ch.input = append(ch.input, input)
	ch.currentPos = len(ch.input)
}

func (ch *chatHistory) navigateHistory(direction int) string {
	if len(ch.input) == 0 {
		return ""
	}

	ch.currentPos += direction

	if ch.currentPos < 0 {
		ch.currentPos = 0
	} else if ch.currentPos >= len(ch.input) {
		ch.currentPos = len(ch.input)
		return ""
	}

	return ch.input[ch.currentPos]
}

func (ch *chatHistory) clear() {
	ch.messages = []string{}
	ch.input = []string{}
	ch.currentPos = 0
}

type Agent interface {
	Process(ctx context.Context, input, prompt string) (string, error)
}

type model struct {
	viewport viewport.Model
	textarea textarea.Model
	history  chatHistory
	err      error
	agent    Agent
	ctx      context.Context
}

func NewModel(ctx context.Context, agent Agent, width, height int) model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	vp := viewport.New(width, height)
	vp.SetContent("Welcome to the chat! Type a message and press Enter to send.")

	return model{
		textarea: ta,
		viewport: vp,
		agent:    agent,
		ctx:      ctx,
		history:  chatHistory{},
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			return m.handleEnter()
		case tea.KeyUp:
			return m.navigateHistory(-1)
		case tea.KeyDown:
			return m.navigateHistory(1)
		case tea.KeyCtrlL:
			return m.clearHistory()
		}
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 4 // Adjust for textarea and status line
		m.textarea.SetWidth(msg.Width - 2) // Subtract 2 for padding
		m.textarea.SetHeight(3)            // Set a fixed height for the textarea
		return m, nil
	}

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m model) handleEnter() (tea.Model, tea.Cmd) {
	if m.textarea.Value() != "" {
		m = m.addUserMessage(m.textarea.Value())
		m = m.processAIResponse(m.textarea.Value())
		m.textarea.Reset()
		m.viewport.GotoBottom()
	}
	return m, nil
}

func (m model) addUserMessage(message string) model {
	m.history.addMessage("You: " + message)
	m.history.addInput(message)
	m.updateViewport()
	return m
}

func (m model) processAIResponse(prompt string) model {
	response, err := m.agent.Process(m.ctx, prompt, "")
	if err != nil {
		m.err = fmt.Errorf("AI processing error: %w", err)
		m.history.addMessage("Error: " + err.Error())
	} else {
		m.history.addMessage("AI: " + response)
	}
	m.updateViewport()
	return m
}

func (m model) navigateHistory(direction int) (tea.Model, tea.Cmd) {
	m.textarea.SetValue(m.history.navigateHistory(direction))
	return m, nil
}

func (m model) clearHistory() (tea.Model, tea.Cmd) {
	m.history.clear()
	m.updateViewport()
	return m, nil
}

func (m *model) updateViewport() {
	m.viewport.SetContent(strings.Join(m.history.messages, "\n"))
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n\n%s",
		m.viewport.View(),
		m.textarea.View(),
	) + "\n\n(ctrl+c to quit, ctrl+l to clear history)"
}

func StartTUI(ctx context.Context, agent Agent, width, height int) error {
	p := tea.NewProgram(NewModel(ctx, agent, width, height), tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}
	return nil
}
