package streaming

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestSSEParser(t *testing.T) {
	t.Run("BasicEvent", func(t *testing.T) {
		input := `data: Hello World

`
		parser := NewSSEParser(strings.NewReader(input))
		event, err := parser.ParseNext(context.Background())

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if event.Data != "Hello World" {
			t.Errorf("Expected data 'Hello World', got: %s", event.Data)
		}
	})

	t.Run("MultilineData", func(t *testing.T) {
		input := `data: Line 1
data: Line 2
data: Line 3

`
		parser := NewSSEParser(strings.NewReader(input))
		event, err := parser.ParseNext(context.Background())

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		expected := "Line 1\nLine 2\nLine 3"
		if event.Data != expected {
			t.Errorf("Expected multiline data, got: %s", event.Data)
		}
	})

	t.Run("CompleteEvent", func(t *testing.T) {
		input := `id: 123
event: message
data: Test data
retry: 5000

`
		parser := NewSSEParser(strings.NewReader(input))
		event, err := parser.ParseNext(context.Background())

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if event.ID != "123" {
			t.Errorf("Expected ID '123', got: %s", event.ID)
		}

		if event.Type != "message" {
			t.Errorf("Expected type 'message', got: %s", event.Type)
		}

		if event.Data != "Test data" {
			t.Errorf("Expected data 'Test data', got: %s", event.Data)
		}

		if event.Retry != 5000 {
			t.Errorf("Expected retry 5000, got: %d", event.Retry)
		}
	})

	t.Run("Comments", func(t *testing.T) {
		input := `: This is a comment
data: Real data

`
		parser := NewSSEParser(strings.NewReader(input))
		event, err := parser.ParseNext(context.Background())

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if event.Comment != " This is a comment" {
			t.Errorf("Expected comment, got: %s", event.Comment)
		}

		if event.Data != "Real data" {
			t.Errorf("Expected data 'Real data', got: %s", event.Data)
		}
	})

	t.Run("MultipleEvents", func(t *testing.T) {
		input := `data: Event 1

data: Event 2

data: Event 3

`
		parser := NewSSEParser(strings.NewReader(input))

		for i := 1; i <= 3; i++ {
			event, err := parser.ParseNext(context.Background())
			if err != nil {
				t.Errorf("Event %d: Expected no error, got: %v", i, err)
			}

			expected := "Event " + string(rune('0'+i))
			if event.Data != expected {
				t.Errorf("Event %d: Expected data '%s', got: %s", i, expected, event.Data)
			}
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		// Test immediate cancellation
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		input := strings.NewReader("data: Test\n\n")
		parser := NewSSEParser(input)

		_, err := parser.ParseNext(ctx)
		if err != context.Canceled {
			t.Errorf("Expected context canceled error, got: %v", err)
		}
	})
}

func TestStreamProcessor(t *testing.T) {
	t.Run("DataHandler", func(t *testing.T) {
		input := `data: Chunk 1

data: Chunk 2

data: Chunk 3

`
		var chunks []string
		processor := NewStreamProcessor(strings.NewReader(input)).
			WithDataHandler(func(data string) error {
				chunks = append(chunks, data)
				return nil
			})

		err := processor.Process(context.Background())
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(chunks) != 3 {
			t.Errorf("Expected 3 chunks, got: %d", len(chunks))
		}

		for i, chunk := range chunks {
			expected := "Chunk " + string(rune('1'+i))
			if chunk != expected {
				t.Errorf("Chunk %d: Expected '%s', got: %s", i, expected, chunk)
			}
		}
	})

	t.Run("EventHandler", func(t *testing.T) {
		input := `id: 1
event: start
data: Starting

id: 2
event: progress
data: Working

id: 3
event: complete
data: Done

`
		var events []*SSEEvent
		processor := NewStreamProcessor(strings.NewReader(input)).
			WithEventHandler(func(event *SSEEvent) error {
				events = append(events, event)
				return nil
			})

		err := processor.Process(context.Background())
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(events) != 3 {
			t.Errorf("Expected 3 events, got: %d", len(events))
		}

		expectedTypes := []string{"start", "progress", "complete"}
		for i, event := range events {
			if event.Type != expectedTypes[i] {
				t.Errorf("Event %d: Expected type '%s', got: %s", i, expectedTypes[i], event.Type)
			}
		}
	})
}

func TestJSONStreamProcessor(t *testing.T) {
	t.Run("JSONData", func(t *testing.T) {
		input := `data: {"type": "start", "message": "Beginning"}

data: {"type": "update", "progress": 50}

data: {"type": "complete", "result": "success"}

data: [DONE]

`
		var messages []map[string]interface{}
		processor := NewJSONStreamProcessor(strings.NewReader(input)).
			WithJSONHandler(func(data json.RawMessage) error {
				var msg map[string]interface{}
				if err := json.Unmarshal(data, &msg); err != nil {
					return err
				}
				messages = append(messages, msg)
				return nil
			})

		err := processor.Process(context.Background())
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if len(messages) != 3 {
			t.Errorf("Expected 3 messages, got: %d", len(messages))
		}

		// Check first message
		if messages[0]["type"] != "start" {
			t.Errorf("Expected first message type 'start', got: %v", messages[0]["type"])
		}

		// Check second message
		if progress, ok := messages[1]["progress"].(float64); !ok || progress != 50 {
			t.Errorf("Expected progress 50, got: %v", messages[1]["progress"])
		}

		// Check third message
		if messages[2]["result"] != "success" {
			t.Errorf("Expected result 'success', got: %v", messages[2]["result"])
		}
	})
}

func TestChunkedTextBuilder(t *testing.T) {
	builder := NewChunkedTextBuilder()

	// Add chunks
	chunks := []string{"Hello", " ", "World", "!", " How", " are", " you?"}
	for _, chunk := range chunks {
		builder.Append(chunk)
	}

	result := builder.String()
	expected := "Hello World! How are you?"

	if result != expected {
		t.Errorf("Expected '%s', got: '%s'", expected, result)
	}

	// Test clear
	builder.Clear()
	if builder.String() != "" {
		t.Error("Expected empty string after clear")
	}

	// Test reuse after clear
	builder.Append("New text")
	if builder.String() != "New text" {
		t.Errorf("Expected 'New text', got: '%s'", builder.String())
	}
}
