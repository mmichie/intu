package ui

import (
	"context"
	"fmt"
	"strings"
	"sync"
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
type StreamHandler = func(chunk string) error

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

// Program reference for message passing
var activeProgramMu sync.Mutex
var activeProgram *tea.Program

type model struct {
	viewport      viewport.Model
	textarea      textarea.Model
	history       chatHistory
	err           error
	agent         Agent
	ctx           context.Context
	loading       bool
	streaming     bool
	inputHeight   int
	statusBar     string
	width         int
	height        int
	spinner       int
	spinnerFrames []string
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
	return tickMsg{}
}

// Create a custom tick message type
type tickMsg struct{}

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
	case tickMsg:
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

	case streamingMsg:
		// Set streaming state based on the message
		m.streaming = !msg.done
		m.loading = !msg.done

		// Handle errors
		if msg.err != nil {
			m.streaming = false
			m.loading = false
			m.err = fmt.Errorf("AI streaming error: %w", msg.err)

			// Check if this is a timeout error
			if strings.Contains(msg.err.Error(), "timed out") {
				// If we have some content, preserve it and add a note
				if msg.content != "" {
					// Content is already formatted by StreamBuffer
					m.history.addMessage("AI: " + msg.content + "\n\n[Response timed out - showing partial content]")
				} else {
					m.history.addMessage("Error: " + msg.err.Error())
				}
			} else {
				m.history.addMessage("Error: " + msg.err.Error())
			}

			m.updateViewport()
			return m, nil
		}

		// Handle completion
		if msg.done {
			// Streaming completed
			m.streaming = false
			m.loading = false

			// Add the complete response to history
			if msg.content != "" {
				// Find and replace the in-progress message if it exists
				// Search from newest to oldest to get the most recent matching message
				lastMsgIndex := -1
				for i := len(m.history.messages) - 1; i >= 0; i-- {
					if strings.HasPrefix(m.history.messages[i], "AI: ") &&
						!strings.Contains(m.history.messages[i], "[Response") {
						lastMsgIndex = i
						break
					}
				}

				if lastMsgIndex >= 0 {
					// Replace the existing message with already formatted content
					m.history.messages[lastMsgIndex] = "AI: " + msg.content
				} else {
					// Add as a new message
					m.history.addMessage("AI: " + msg.content)
				}
			}

			m.updateViewport()
			return m, nil
		}

		// This is an intermediate chunk update
		if msg.content == "" {
			// Skip empty updates
			return m, nil
		}

		// Find the existing message if any, searching from the end of history
		// to ensure we update the most recent message
		lastMsgIndex := -1
		for i := len(m.history.messages) - 1; i >= 0; i-- {
			if strings.HasPrefix(m.history.messages[i], "AI: ") &&
				!strings.Contains(m.history.messages[i], "[Response") {
				lastMsgIndex = i
				break
			}
		}

		if lastMsgIndex >= 0 {
			// Update the existing message with the already formatted content
			m.history.messages[lastMsgIndex] = "AI: " + msg.content
		} else {
			// No message found, create a new one
			m.history.addMessage("AI: " + msg.content)
		}

		m.updateViewport()
		return m, nil

	case tea.KeyMsg:
		// Allow keyboard input during streaming - only block during initial loading
		// Always allow quit commands
		if (m.loading && !m.streaming) &&
			msg.Type != tea.KeyCtrlC &&
			msg.Type != tea.KeyCtrlD &&
			msg.Type != tea.KeyEsc {
			return m, nil
		}
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyCtrlD, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.textarea.Value() != "" {
				// Handle active streaming differently from just loading
				if m.streaming {
					// Only show interruption message if actually streaming content
					m.streaming = false
					m.loading = false

					// Find the streaming message and mark it as interrupted
					for i, message := range m.history.messages {
						if strings.HasPrefix(message, "AI: ") && !strings.Contains(message, "[Response") {
							m.history.messages[i] = message + "\n[Response interrupted by new request]"
							break
						}
					}
				} else if m.loading {
					// If just loading but not streaming content yet, simply reset states
					m.streaming = false
					m.loading = false
				}

				prompt := m.textarea.Value()
				m.history.addMessage("You: " + prompt)
				m.history.addInput(prompt)
				m.textarea.Reset()
				m.updateViewport()
				m.loading = true

				// Always use non-streaming mode for better stability
				return m, func() tea.Msg {
					// Use a timeout context for the request
					timeoutCtx, cancel := context.WithTimeout(m.ctx, 90*time.Second)
					defer cancel()

					// Process the request with a timeout
					response, err := m.agent.Process(timeoutCtx, prompt, "")
					if err == nil {
						response = FormatMarkdown(cleanResponse(response))
					}
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
	// Join messages with a single newline for cleaner rendering
	// This creates a more compact chat history appearance
	content := strings.Join(m.history.messages, "\n")

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

// StartTUI initializes and starts the text user interface
func StartTUI(ctx context.Context, agent Agent, width, height int) error {
	// Create new program
	p := tea.NewProgram(
		NewModel(ctx, agent, width, height),
		tea.WithAltScreen(),
	)

	// Store program reference for message passing
	activeProgramMu.Lock()
	activeProgram = p
	activeProgramMu.Unlock()

	// Run the program
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}
