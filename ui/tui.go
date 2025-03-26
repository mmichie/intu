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

// Message for AI responses
type aiResponseMsg struct {
	response string
	err      error
}

type model struct {
	viewport    viewport.Model
	textarea    textarea.Model
	history     chatHistory
	err         error
	agent       Agent
	ctx         context.Context
	loading     bool
	inputHeight int
	statusBar   string
	width       int
	height      int
}

func NewModel(ctx context.Context, agent Agent, width, height int) model {
	inputHeight := 3
	statusHeight := 1
	vpHeight := height - inputHeight - statusHeight - 2 // 2 for padding

	if vpHeight < 10 {
		vpHeight = 10 // Minimum reasonable height
	}

	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Focus()
	ta.SetHeight(inputHeight)
	ta.SetWidth(width - 2) // 2 for padding

	vp := viewport.New(width, vpHeight)

	// Initialize chat history
	history := chatHistory{}
	history.addMessage("Welcome to the chat! Type a message and press Enter to send.")

	m := model{
		viewport:    vp,
		textarea:    ta,
		agent:       agent,
		ctx:         ctx,
		history:     history,
		inputHeight: inputHeight,
		statusBar:   "(ctrl+c/ctrl+d to quit, ctrl+l to clear history)",
		width:       width,
		height:      height,
	}

	// Initialize viewport content
	m.updateViewport()

	return m
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
	case aiResponseMsg:
		m.loading = false
		if msg.err != nil {
			m.err = fmt.Errorf("AI processing error: %w", msg.err)
			m.history.addMessage("Error: " + msg.err.Error())
		} else {
			m.history.addMessage("AI: " + msg.response)
		}
		m.updateViewport()
		return m, nil

	case tea.KeyMsg:
		// Block keyboard input while loading, except for Ctrl+C or Ctrl+D to quit
		if m.loading && msg.Type != tea.KeyCtrlC && msg.Type != tea.KeyCtrlD {
			return m, nil
		}
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlD, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.textarea.Value() != "" {
				prompt := m.textarea.Value()
				m.history.addMessage("You: " + prompt)
				m.history.addInput(prompt)
				m.textarea.Reset()
				m.updateViewport()
				m.loading = true
				return m, func() tea.Msg {
					response, err := m.agent.Process(m.ctx, prompt, "")
					return aiResponseMsg{
						response: response,
						err:      err,
					}
				}
			}
			return m, nil
		case tea.KeyUp:
			m.textarea.SetValue(m.history.navigateHistory(-1))
			return m, nil
		case tea.KeyDown:
			m.textarea.SetValue(m.history.navigateHistory(1))
			return m, nil
		case tea.KeyCtrlL:
			m.history.clear()
			m.updateViewport()
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		inputHeight := 3
		statusHeight := 1
		vpHeight := msg.Height - inputHeight - statusHeight - 2 // 2 for padding

		if vpHeight < 5 {
			vpHeight = 5
		}

		m.viewport.Width = msg.Width
		m.viewport.Height = vpHeight

		m.textarea.SetWidth(msg.Width - 2)
		m.textarea.SetHeight(inputHeight)

		m.updateViewport()
		return m, nil
	}

	// Handle textarea and viewport updates
	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m *model) updateViewport() {
	content := strings.Join(m.history.messages, "\n\n")
	m.viewport.SetContent(content)

	// Make sure we're scrolled to the bottom
	if len(m.history.messages) > 0 {
		m.viewport.GotoBottom()
	}
}

func (m model) View() string {
	statusBar := m.statusBar
	if m.loading {
		statusBar = "Loading response... " + statusBar
	}

	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		m.viewport.View(),
		m.textarea.View(),
		statusBar,
	)
}

func StartTUI(ctx context.Context, agent Agent, width, height int) error {
	p := tea.NewProgram(
		NewModel(ctx, agent, width, height),
		tea.WithAltScreen(),
	)
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}
	return nil
}
