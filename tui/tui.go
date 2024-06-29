package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	viewport   viewport.Model
	textarea   textarea.Model
	messages   []string
	history    []string
	historyPos int
	err        error
	aiClient   AIClient
	ctx        context.Context
}

type AIClient interface {
	ProcessWithAI(ctx context.Context, input, prompt string) (string, error)
}

func NewModel(ctx context.Context, aiClient AIClient) model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	return model{
		textarea:   ta,
		viewport:   viewport.New(80, 20),
		aiClient:   aiClient,
		ctx:        ctx,
		history:    make([]string, 0),
		historyPos: -1,
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

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

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
		}
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m model) handleEnter() (tea.Model, tea.Cmd) {
	if m.textarea.Value() != "" {
		m.messages = append(m.messages, "You: "+m.textarea.Value())
		m.history = append(m.history, m.textarea.Value())
		m.historyPos = len(m.history)

		response, err := m.aiClient.ProcessWithAI(m.ctx, m.textarea.Value(), m.textarea.Value())
		if err != nil {
			m.err = err
		} else {
			m.messages = append(m.messages, "AI: "+response)
		}
		m.viewport.SetContent(strings.Join(m.messages, "\n"))
		m.textarea.Reset()
		m.viewport.GotoBottom()
	}
	return m, nil
}

func (m model) navigateHistory(direction int) (tea.Model, tea.Cmd) {
	if len(m.history) == 0 {
		return m, nil
	}

	m.historyPos += direction

	if m.historyPos < 0 {
		m.historyPos = 0
	} else if m.historyPos >= len(m.history) {
		m.historyPos = len(m.history)
		m.textarea.SetValue("")
	} else {
		m.textarea.SetValue(m.history[m.historyPos])
	}

	return m, nil
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n\n%s",
		m.viewport.View(),
		m.textarea.View(),
	) + "\n\n(ctrl+c to quit)"
}

func StartTUI(ctx context.Context, aiClient AIClient) error {
	p := tea.NewProgram(NewModel(ctx, aiClient))
	_, err := p.Run()
	return err
}
