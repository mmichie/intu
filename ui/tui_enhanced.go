package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

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

// StreamHandler is a function that handles streaming text chunks
type StreamHandler func(chunk string) error

// Agent interface defines the methods required for an AI agent in the UI
type Agent interface {
	// Process sends a request and returns the full response
	Process(ctx context.Context, input, prompt string) (string, error)

	// SupportsStreaming indicates if the agent supports streaming responses
	SupportsStreaming() bool

	// ProcessStreaming sends a request and streams the response chunks
	ProcessStreaming(ctx context.Context, input, prompt string, handler StreamHandler) error
}

// Message types for AI interactions
type aiResponseMsg struct {
	response string
	err      error
}

// Streaming chunk message
type aiStreamChunkMsg struct {
	chunk string
	done  bool
	err   error
}

type model struct {
	viewport        viewport.Model
	textarea        textarea.Model
	history         chatHistory
	err             error
	agent           Agent
	ctx             context.Context
	loading         bool
	streaming       bool
	currentResponse string
	inputHeight     int
	statusBar       string
	width           int
	height          int
	spinner         int
	spinnerFrames   []string
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

	// Define spinner frames for loading animation
	spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	m := model{
		viewport:      vp,
		textarea:      ta,
		agent:         agent,
		ctx:           ctx,
		history:       history,
		inputHeight:   inputHeight,
		statusBar:     "(ctrl+c/ctrl+d to quit, ctrl+l to clear history)",
		width:         width,
		height:        height,
		spinner:       0,
		spinnerFrames: spinnerFrames,
	}

	// Initialize viewport content
	m.updateViewport()

	return m
}

// spinnerTick is a command that updates the spinner
func spinnerTick() tea.Msg {
	return tea.KeyTick
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		tea.Every(time.Millisecond*100, func(t time.Time) tea.Msg {
			return spinnerTick()
		}),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyTick:
		// Update spinner for loading animation
		if m.loading {
			m.updateSpinner()
			return m, nil
		}
	case aiResponseMsg:
		m.loading = false
		m.streaming = false
		if msg.err != nil {
			m.err = fmt.Errorf("AI processing error: %w", msg.err)
			m.history.addMessage("Error: " + msg.err.Error())
		} else {
			m.history.addMessage("AI: " + msg.response)
		}
		m.updateViewport()
		return m, nil

	case aiStreamChunkMsg:
		if msg.err != nil {
			m.streaming = false
			m.loading = false
			m.err = fmt.Errorf("AI streaming error: %w", msg.err)
			m.history.addMessage("Error: " + msg.err.Error())
			m.updateViewport()
			return m, nil
		}

		if msg.done {
			// Streaming completed
			m.streaming = false
			m.loading = false
			// Add the complete response to history
			if m.currentResponse != "" {
				m.history.addMessage("AI: " + m.currentResponse)
				m.currentResponse = ""
			}
			m.updateViewport()
			return m, nil
		}

		// Append chunk to current response and update viewport
		if !m.streaming {
			// First chunk, initialize streaming
			m.streaming = true
			m.currentResponse = msg.chunk
			m.history.addMessage("AI: " + m.currentResponse)
		} else {
			// Append to existing response
			m.currentResponse += msg.chunk
			// Update the last message
			lastMsgIndex := len(m.history.messages) - 1
			if lastMsgIndex >= 0 && strings.HasPrefix(m.history.messages[lastMsgIndex], "AI: ") {
				m.history.messages[lastMsgIndex] = "AI: " + m.currentResponse
			}
		}

		m.updateViewport()
		// Continue streaming animation
		return m, nil

	case tea.KeyMsg:
		// Block keyboard input while loading or streaming, except for Ctrl+C or Ctrl+D to quit
		if (m.loading || m.streaming) && msg.Type != tea.KeyCtrlC && msg.Type != tea.KeyCtrlD {
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

				// Check if agent supports streaming
				if m.agent.SupportsStreaming() {
					// Start streaming
					return m, func() tea.Msg {
						var streamErr error

						// Create a handler for streaming chunks
						handleChunk := func(chunk string) error {
							// Queue a message for the UI to process the chunk
							cmd := sendStreamingCommand(chunk, false, nil)
							tea.ExecCommand(cmd)()
							return nil
						}

						// Process the request with streaming
						streamErr = m.agent.ProcessStreaming(m.ctx, prompt, "", handleChunk)

						// Send a final message indicating streaming is done
						return aiStreamChunkMsg{
							chunk: "",
							done:  true,
							err:   streamErr,
						}
					}
				} else {
					// Fallback to non-streaming
					return m, func() tea.Msg {
						response, err := m.agent.Process(m.ctx, prompt, "")
						return aiResponseMsg{
							response: response,
							err:      err,
						}
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

// spinnerFrame gets the current spinner frame
func (m model) spinnerFrame() string {
	if len(m.spinnerFrames) == 0 {
		return ""
	}
	return m.spinnerFrames[m.spinner%len(m.spinnerFrames)]
}

// updateSpinner advances the spinner animation
func (m *model) updateSpinner() {
	m.spinner++
}

func (m model) View() string {
	statusBar := m.statusBar
	if m.loading {
		if m.streaming {
			statusBar = fmt.Sprintf("%s Streaming response... %s", m.spinnerFrame(), statusBar)
		} else {
			statusBar = fmt.Sprintf("%s Loading response... %s", m.spinnerFrame(), statusBar)
		}
	}

	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		m.viewport.View(),
		m.textarea.View(),
		statusBar,
	)
}

// sendStreamingCommand creates a tea.Cmd that sends streaming chunks via a channel
func sendStreamingCommand(chunk string, done bool, err error) tea.Cmd {
	return func() tea.Msg {
		return aiStreamChunkMsg{
			chunk: chunk,
			done:  done,
			err:   err,
		}
	}
}

// StartTUIEnhanced starts an enhanced TUI with streaming support
func StartTUIEnhanced(ctx context.Context, agent Agent, width, height int) error {
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
