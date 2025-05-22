// Package streaming provides utilities for handling streaming responses
package streaming

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	ID      string
	Type    string
	Data    string
	Retry   int
	Comment string
}

// SSEParser handles parsing of Server-Sent Events streams
type SSEParser struct {
	reader      *bufio.Reader
	maxLineSize int
	timeout     time.Duration
}

// NewSSEParser creates a new SSE parser
func NewSSEParser(reader io.Reader) *SSEParser {
	return &SSEParser{
		reader:      bufio.NewReader(reader),
		maxLineSize: 1024 * 1024, // 1MB max line size
		timeout:     30 * time.Second,
	}
}

// WithMaxLineSize sets the maximum line size for the parser
func (p *SSEParser) WithMaxLineSize(size int) *SSEParser {
	p.maxLineSize = size
	return p
}

// WithTimeout sets the timeout for reading events
func (p *SSEParser) WithTimeout(timeout time.Duration) *SSEParser {
	p.timeout = timeout
	return p
}

// ParseNext reads and parses the next SSE event from the stream
func (p *SSEParser) ParseNext(ctx context.Context) (*SSEEvent, error) {
	event := &SSEEvent{}
	var currentField string
	var currentValue strings.Builder

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		line, err := p.readLine()
		if err != nil {
			if err == io.EOF && currentField != "" {
				// Process the last field if we hit EOF
				p.processField(event, currentField, currentValue.String())
				if event.Data != "" || event.ID != "" || event.Type != "" {
					return event, nil
				}
			}
			return nil, err
		}

		// Empty line signals end of event
		if line == "" {
			if currentField != "" {
				p.processField(event, currentField, currentValue.String())
			}

			// Only return event if it has data
			if event.Data != "" || event.ID != "" || event.Type != "" {
				return event, nil
			}

			// Reset for next event
			currentField = ""
			currentValue.Reset()
			continue
		}

		// Comment line
		if strings.HasPrefix(line, ":") {
			event.Comment = strings.TrimPrefix(line, ":")
			continue
		}

		// Field line
		colonIndex := strings.Index(line, ":")
		if colonIndex == -1 {
			// Line with just field name, no value
			currentField = line
			currentValue.Reset()
		} else {
			fieldName := line[:colonIndex]
			fieldValue := strings.TrimPrefix(line[colonIndex+1:], " ")

			if currentField == fieldName {
				// Continue building multi-line value
				currentValue.WriteString("\n")
				currentValue.WriteString(fieldValue)
			} else {
				// New field, process previous one
				if currentField != "" {
					p.processField(event, currentField, currentValue.String())
				}
				currentField = fieldName
				currentValue.Reset()
				currentValue.WriteString(fieldValue)
			}
		}
	}
}

// readLine reads a line from the reader with size limit
func (p *SSEParser) readLine() (string, error) {
	line, isPrefix, err := p.reader.ReadLine()
	if err != nil {
		return "", err
	}

	// Check if line exceeds max size
	if isPrefix {
		// Discard the rest of the line
		for isPrefix {
			_, isPrefix, err = p.reader.ReadLine()
			if err != nil {
				return "", err
			}
		}
		return "", errors.New("line exceeds maximum size")
	}

	return string(line), nil
}

// processField updates the event based on field name and value
func (p *SSEParser) processField(event *SSEEvent, field, value string) {
	switch field {
	case "id":
		event.ID = value
	case "event":
		event.Type = value
	case "data":
		if event.Data != "" {
			event.Data += "\n" + value
		} else {
			event.Data = value
		}
	case "retry":
		// Parse retry as integer milliseconds
		if retry, err := parseInt(value); err == nil {
			event.Retry = retry
		}
	}
}

// parseInt safely parses an integer
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// StreamProcessor handles processing of SSE streams with custom handlers
type StreamProcessor struct {
	parser       *SSEParser
	dataHandler  func(data string) error
	eventHandler func(event *SSEEvent) error
	errorHandler func(err error) error
}

// NewStreamProcessor creates a new stream processor
func NewStreamProcessor(reader io.Reader) *StreamProcessor {
	return &StreamProcessor{
		parser: NewSSEParser(reader),
	}
}

// WithDataHandler sets a handler for data fields
func (sp *StreamProcessor) WithDataHandler(handler func(string) error) *StreamProcessor {
	sp.dataHandler = handler
	return sp
}

// WithEventHandler sets a handler for complete events
func (sp *StreamProcessor) WithEventHandler(handler func(*SSEEvent) error) *StreamProcessor {
	sp.eventHandler = handler
	return sp
}

// WithErrorHandler sets a handler for errors
func (sp *StreamProcessor) WithErrorHandler(handler func(error) error) *StreamProcessor {
	sp.errorHandler = handler
	return sp
}

// Process processes the entire stream
func (sp *StreamProcessor) Process(ctx context.Context) error {
	for {
		event, err := sp.parser.ParseNext(ctx)
		if err != nil {
			if err == io.EOF {
				return nil
			}

			if sp.errorHandler != nil {
				if handlerErr := sp.errorHandler(err); handlerErr != nil {
					return handlerErr
				}
				continue
			}
			return err
		}

		// Skip empty events
		if event.Data == "" && event.Type == "" && event.ID == "" {
			continue
		}

		// Handle complete event
		if sp.eventHandler != nil {
			if err := sp.eventHandler(event); err != nil {
				return err
			}
		}

		// Handle data specifically
		if event.Data != "" && sp.dataHandler != nil {
			if err := sp.dataHandler(event.Data); err != nil {
				return err
			}
		}
	}
}

// JSONStreamProcessor processes SSE streams containing JSON data
type JSONStreamProcessor struct {
	*StreamProcessor
	jsonHandler func(data json.RawMessage) error
}

// NewJSONStreamProcessor creates a processor for JSON SSE streams
func NewJSONStreamProcessor(reader io.Reader) *JSONStreamProcessor {
	sp := NewStreamProcessor(reader)
	jsp := &JSONStreamProcessor{
		StreamProcessor: sp,
	}

	// Set up data handler to parse JSON
	sp.WithDataHandler(func(data string) error {
		// Skip special markers
		if data == "[DONE]" || data == "" {
			return nil
		}

		// Try to parse as JSON
		if jsp.jsonHandler != nil {
			return jsp.jsonHandler(json.RawMessage(data))
		}
		return nil
	})

	return jsp
}

// WithJSONHandler sets a handler for parsed JSON data
func (jsp *JSONStreamProcessor) WithJSONHandler(handler func(json.RawMessage) error) *JSONStreamProcessor {
	jsp.jsonHandler = handler
	return jsp
}

// ChunkedTextBuilder helps build text from streaming chunks
type ChunkedTextBuilder struct {
	chunks []string
	total  int
}

// NewChunkedTextBuilder creates a new text builder
func NewChunkedTextBuilder() *ChunkedTextBuilder {
	return &ChunkedTextBuilder{
		chunks: make([]string, 0, 100),
	}
}

// Append adds a chunk to the builder
func (ctb *ChunkedTextBuilder) Append(chunk string) {
	ctb.chunks = append(ctb.chunks, chunk)
	ctb.total += len(chunk)
}

// String returns the complete text
func (ctb *ChunkedTextBuilder) String() string {
	if len(ctb.chunks) == 0 {
		return ""
	}

	// Pre-allocate capacity for efficiency
	var builder strings.Builder
	builder.Grow(ctb.total)

	for _, chunk := range ctb.chunks {
		builder.WriteString(chunk)
	}

	return builder.String()
}

// Clear resets the builder
func (ctb *ChunkedTextBuilder) Clear() {
	ctb.chunks = ctb.chunks[:0]
	ctb.total = 0
}
