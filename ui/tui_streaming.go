package ui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// StreamBuffer represents a thread-safe buffer for collecting and managing streaming content
type StreamBuffer struct {
	mu           sync.Mutex
	fullText     string
	inCodeBlock  bool
	complete     bool
	chunks       []string
	lastOverlaps map[string]bool
}

// NewStreamBuffer creates a new streaming buffer
func NewStreamBuffer() *StreamBuffer {
	return &StreamBuffer{
		fullText:     "",
		inCodeBlock:  false,
		complete:     false,
		chunks:       []string{},
		lastOverlaps: make(map[string]bool),
	}
}

// AddChunk safely adds a chunk to the buffer
func (sb *StreamBuffer) AddChunk(chunk string) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	if chunk == "" {
		return
	}

	// Clean up Claude-specific JSON formatting if needed
	chunk = cleanupChunk(chunk)

	// Check for duplicate content
	if sb.isDuplicate(chunk) {
		return
	}

	// Track chunk for future duplication checks
	sb.chunks = append(sb.chunks, chunk)
	if len(sb.chunks) > 50 {
		// Keep only the last 50 chunks to avoid memory growth
		sb.chunks = sb.chunks[len(sb.chunks)-50:]
	}

	// Check for code block state changes
	if strings.Contains(chunk, "```") {
		// Count occurrences
		count := strings.Count(chunk, "```")
		// Toggle state for each marker
		for i := 0; i < count; i++ {
			sb.inCodeBlock = !sb.inCodeBlock
		}
	}

	// Append the chunk with intelligent overlap detection
	sb.fullText = sb.appendWithOverlapDetection(sb.fullText, chunk)
}

// isDuplicate checks if a chunk is a duplicate or already contained
func (sb *StreamBuffer) isDuplicate(chunk string) bool {
	// Skip if chunk is already in our full text
	if strings.Contains(sb.fullText, chunk) {
		return true
	}

	// Look for exact matches in recent chunks
	for _, prevChunk := range sb.chunks {
		if chunk == prevChunk {
			return true
		}
	}

	// Look for near-duplicates - chunks that differ only in whitespace
	normalizedChunk := normalizeWhitespace(chunk)
	if normalizedChunk == "" {
		return true
	}

	for _, prevChunk := range sb.chunks {
		if normalizeWhitespace(prevChunk) == normalizedChunk {
			return true
		}
	}

	return false
}

// normalizeWhitespace collapses whitespace for comparison
func normalizeWhitespace(s string) string {
	// Replace all whitespace sequences with a single space
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}

// appendWithOverlapDetection intelligently appends chunks with overlap detection
func (sb *StreamBuffer) appendWithOverlapDetection(currentText, newChunk string) string {
	// First chunk or empty current text
	if currentText == "" {
		return newChunk
	}

	// In code blocks, preserve exact formatting
	if sb.inCodeBlock {
		// Check for overlaps even in code blocks
		for overlapSize := 20; overlapSize >= 3; overlapSize-- {
			if len(currentText) >= overlapSize && len(newChunk) >= overlapSize {
				endOfCurrent := currentText[len(currentText)-overlapSize:]
				startOfNew := newChunk[:overlapSize]

				if endOfCurrent == startOfNew {
					// Remember this overlap to avoid duplicating it again
					sb.lastOverlaps[startOfNew] = true
					return currentText + newChunk[overlapSize:]
				}
			}
		}

		// If no overlap found in code block, just append
		return currentText + newChunk
	}

	// For normal text, check for overlaps
	for overlapSize := 30; overlapSize >= 3; overlapSize-- {
		if len(currentText) >= overlapSize && len(newChunk) >= overlapSize {
			endOfCurrent := currentText[len(currentText)-overlapSize:]
			startOfNew := newChunk[:overlapSize]

			if endOfCurrent == startOfNew {
				// Remember this overlap to avoid duplicating it again
				sb.lastOverlaps[startOfNew] = true
				return currentText + newChunk[overlapSize:]
			}
		}
	}

	// Check for doubled content (same content appears twice in a row)
	halfLen := len(newChunk)
	if halfLen > 3 && len(currentText) >= halfLen {
		endOfCurrent := currentText[len(currentText)-halfLen:]
		if endOfCurrent == newChunk {
			return currentText // Skip entirely as it's a duplicate
		}
	}

	// Regular text - check if we need a space
	if len(currentText) > 0 && len(newChunk) > 0 {
		lastChar := rune(currentText[len(currentText)-1])
		firstChar := rune(newChunk[0])

		// Add a space between words if needed
		if isWordChar(lastChar) && isWordChar(firstChar) &&
			!strings.HasSuffix(currentText, " ") &&
			!strings.HasPrefix(newChunk, " ") &&
			!strings.HasSuffix(currentText, "\n") &&
			!strings.HasPrefix(newChunk, "\n") {
			return currentText + " " + newChunk
		}
	}

	// Default case - direct append
	return currentText + newChunk
}

// GetContent returns a copy of the current content
func (sb *StreamBuffer) GetContent() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	// Apply final cleanup to the full text
	cleanedText := removeDuplicates(sb.fullText)
	return cleanedText
}

// GetRawContent returns the raw buffer content without cleanup
func (sb *StreamBuffer) GetRawContent() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.fullText
}

// MarkComplete marks the stream as complete
func (sb *StreamBuffer) MarkComplete() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.complete = true
}

// IsComplete returns if the stream is complete
func (sb *StreamBuffer) IsComplete() bool {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.complete
}

// isWordChar returns true if the character is part of a word
func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// isInCodeBlock checks if we're currently inside a markdown code block
func isInCodeBlock(content string) bool {
	// Count the number of code block markers (```)
	count := strings.Count(content, "```")

	// If we have an odd number of markers, we're inside a code block
	return count%2 == 1
}

// cleanupChunk extracts text from Claude JSON responses and cleans it
func cleanupChunk(chunk string) string {
	// Remove common Claude streaming artifacts
	chunk = strings.TrimPrefix(chunk, "data: ")

	// Try to extract pure text from JSON if it seems to be JSON
	if strings.HasPrefix(chunk, "{") && strings.HasSuffix(chunk, "}") {
		// This looks like a JSON object

		// Check for content_block_delta format (Claude API)
		if strings.Contains(chunk, "\"delta\"") && strings.Contains(chunk, "\"text\"") {
			// Extract text from the delta object
			textStart := strings.Index(chunk, "\"text\"")
			if textStart > 0 {
				// Find opening quote after "text":
				quotePos := strings.Index(chunk[textStart+6:], "\"")
				if quotePos > 0 {
					// Find closing quote
					textStart = textStart + 6 + quotePos + 1 // Position after opening quote
					textEnd := strings.Index(chunk[textStart:], "\"")
					if textEnd > 0 {
						chunk = chunk[textStart : textStart+textEnd]
					}
				}
			}
		} else if strings.Contains(chunk, "\"content_block\"") && strings.Contains(chunk, "\"text\"") {
			// Extract text from content_block object (initial block)
			textStart := strings.Index(chunk, "\"text\"")
			if textStart > 0 {
				// Find opening quote after "text":
				quotePos := strings.Index(chunk[textStart+6:], "\"")
				if quotePos > 0 {
					// Find closing quote
					textStart = textStart + 6 + quotePos + 1 // Position after opening quote
					textEnd := strings.Index(chunk[textStart:], "\"")
					if textEnd > 0 {
						chunk = chunk[textStart : textStart+textEnd]
					}
				}
			}
		}
	}

	// Handle any JSON escape sequences
	chunk = strings.ReplaceAll(chunk, "\\n", "\n")
	chunk = strings.ReplaceAll(chunk, "\\\"", "\"")
	chunk = strings.ReplaceAll(chunk, "\\\\", "\\")

	return chunk
}

// streamingMsg represents a streaming update
type streamingMsg struct {
	content string
	done    bool
	err     error
}

// performStreamingRequest executes a streaming request with proper buffer management
func performStreamingRequest(agent Agent, ctx context.Context, prompt string) tea.Cmd {
	return func() tea.Msg {
		// Create a new context with timeout
		timeoutCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
		defer cancel()

		// Create a stream buffer to collect chunks
		buffer := NewStreamBuffer()

		// Create channels for communication
		errChan := make(chan error, 1)
		doneChan := make(chan struct{})

		// Track activity and silence for better completion detection
		var lastActivityTime time.Time
		var lastContentLen int
		completionCheckTicker := time.NewTicker(2 * time.Second)
		defer completionCheckTicker.Stop()

		// Launch a goroutine to check for completion by silence
		go func() {
			lastActivityTime = time.Now()
			lastContentLen = 0

			for {
				select {
				case <-completionCheckTicker.C:
					currentContent := buffer.GetRawContent()
					currentLen := len(currentContent)

					// If we have content and it hasn't changed in a while
					if currentLen > 0 && currentLen == lastContentLen &&
						time.Since(lastActivityTime) > 3*time.Second &&
						!buffer.IsComplete() {

						// Check if the content looks like a complete response
						if isLikelyComplete(currentContent) {
							// It's likely complete, mark it and send final update
							buffer.MarkComplete()
							content := buffer.GetContent()
							content = cleanResponseArtifacts(content)
							finalContent := FormatMarkdown(content)
							sendUIUpdate(finalContent, true, nil)
							return
						}
					}

					lastContentLen = currentLen
				case <-doneChan:
					// Processing completed normally
					return
				case <-timeoutCtx.Done():
					// Context canceled or timed out
					return
				}
			}
		}()

		// Handler for processing streaming chunks
		handleChunk := func(chunk string) error {
			// Check for context cancellation
			select {
			case <-timeoutCtx.Done():
				return timeoutCtx.Err()
			default:
				// Continue processing
			}

			// Update activity timestamp
			lastActivityTime = time.Now()

			// Check for completion markers in the chunk
			if strings.Contains(chunk, "[DONE]") ||
				strings.Contains(chunk, "data: [DONE]") {
				// LLM signals completion
				buffer.MarkComplete()
				content := buffer.GetContent()
				content = cleanResponseArtifacts(content)
				finalContent := FormatMarkdown(content)
				sendUIUpdate(finalContent, true, nil)
				return nil
			}

			// Add the chunk to our buffer
			buffer.AddChunk(chunk)

			// Get a properly formatted version of the content
			formattedContent := buffer.GetContent()

			// Check if this chunk completes the response
			if isLikelyComplete(buffer.GetRawContent()) {
				// It's a natural completion, mark as done
				buffer.MarkComplete()
				content := buffer.GetContent()
				content = cleanResponseArtifacts(content)
				finalContent := FormatMarkdown(content)
				sendUIUpdate(finalContent, true, nil)
				return nil
			}

			// Send intermediate update to UI
			sendUIUpdate(formattedContent, false, nil)

			return nil
		}

		// Start processing in a goroutine
		go func() {
			defer close(doneChan)

			err := agent.ProcessStreaming(timeoutCtx, prompt, "", handleChunk)
			if err != nil {
				errChan <- err
				return
			}

			// Mark the buffer as complete
			buffer.MarkComplete()

			// Send final update with formatting applied
			content := buffer.GetContent()
			content = cleanResponseArtifacts(content)
			finalContent := FormatMarkdown(content)
			sendUIUpdate(finalContent, true, nil)
		}()

		// Set a backup timeout
		go func() {
			select {
			case <-doneChan:
				// Processing completed normally
				return
			case <-time.After(95 * time.Second):
				// Timed out - force completion with what we have
				buffer.MarkComplete()
				errChan <- fmt.Errorf("streaming timed out after 95 seconds")
			}
		}()

		// Wait for first update or completion
		select {
		case err := <-errChan:
			// Error occurred
			if err != nil && strings.Contains(err.Error(), "timed out") {
				// If timeout, return what we have
				content := buffer.GetContent()
				content = cleanResponseArtifacts(content)
				finalContent := FormatMarkdown(content)
				return streamingMsg{
					content: finalContent,
					done:    true,
					err:     err,
				}
			}
			content := buffer.GetContent()
			content = cleanResponseArtifacts(content)
			finalContent := FormatMarkdown(content)
			return streamingMsg{
				content: finalContent,
				done:    true,
				err:     err,
			}
		case <-doneChan:
			// Processing completed
			content := buffer.GetContent()
			content = cleanResponseArtifacts(content)
			finalContent := FormatMarkdown(content)
			return streamingMsg{
				content: finalContent,
				done:    true,
				err:     nil,
			}
		case <-time.After(500 * time.Millisecond):
			// Send an initial update after a short delay even if no chunks received
			return streamingMsg{
				content: buffer.GetContent(),
				done:    false,
				err:     nil,
			}
		}
	}
}

// Helper function to send updates to the UI
func sendUIUpdate(content string, done bool, err error) {
	// Get the active program
	activeProgramMu.Lock()
	p := activeProgram
	activeProgramMu.Unlock()

	if p != nil {
		// Send update to program
		p.Send(streamingMsg{
			content: content,
			done:    done,
			err:     err,
		})
	}
}

// streamingRequestCmd creates a streaming request command
func streamingRequestCmd(agent Agent, ctx context.Context, prompt string) tea.Cmd {
	return performStreamingRequest(agent, ctx, prompt)
}

// isLikelyComplete checks if the content appears to be a completed response
func isLikelyComplete(content string) bool {
	// Common completion phrases
	completionPhrases := []string{
		"How can I assist you",
		"How can I help you",
		"Is there anything else",
		"Is there anything I can help",
		"Let me know if you have any questions",
		"Please let me know if you need",
	}

	// Check if the content ends with any of these phrases
	for _, phrase := range completionPhrases {
		if strings.Contains(content, phrase) {
			return true
		}
	}

	// Check if the content ends with a question mark or period
	trimmedContent := strings.TrimSpace(content)
	if len(trimmedContent) > 0 {
		lastChar := trimmedContent[len(trimmedContent)-1]
		if lastChar == '?' || lastChar == '.' || lastChar == '!' {
			// If it's a proper sentence ending and longer than 20 chars, likely complete
			if len(trimmedContent) > 20 {
				return true
			}
		}
	}

	return false
}
