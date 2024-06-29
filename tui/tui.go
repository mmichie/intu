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
	viewport viewport.Model
	textarea textarea.Model
	messages []string
	err      error
	aiClient AIClient
	ctx      context.Context
}

type AIClient interface {
	ProcessWithAI(ctx context.Context, input, prompt string) (string, error)
}

func NewModel(ctx context.Context, aiClient AIClient) model {
	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()

	return model{
		textarea: ta,
		viewport: viewport.New(80, 20),
		aiClient: aiClient,
		ctx:      ctx,
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
			if m.textarea.Value() != "" {
				m.messages = append(m.messages, "You: "+m.textarea.Value())
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
		}
	}

	return m, tea.Batch(tiCmd, vpCmd)
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
