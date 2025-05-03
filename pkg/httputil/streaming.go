package httputil

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// SendStreamingRequest sends a streaming request to the specified URL
// and processes the response with the stream handler
func SendStreamingRequest(
	ctx context.Context,
	details RequestDetails,
	options ClientOptions,
	handler StreamChunkHandler,
) error {
	if !details.Stream {
		return fmt.Errorf("streaming requested but Stream flag not set in RequestDetails")
	}

	// Create the request
	req, err := createRequest(ctx, details)
	if err != nil {
		return err
	}

	// Set headers specific to streaming requests
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	clientOnce.Do(initClient)

	// Execute the request with retry logic
	for attempt := 0; attempt <= options.RetryAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(options.RetryDelay)
			log.Printf("Retrying streaming request to %s (attempt %d/%d)", req.URL, attempt+1, options.RetryAttempts+1)
		}

		// Create a timeout context for this attempt
		attemptCtx, cancel := context.WithTimeout(ctx, options.Timeout)
		defer cancel()

		reqWithTimeout := req.WithContext(attemptCtx)

		// Send the request
		resp, err := httpClient.Do(reqWithTimeout)
		if err != nil {
			log.Printf("Attempt %d: error sending streaming request: %v", attempt+1, err)
			continue
		}

		// Check if the response is successful
		if resp.StatusCode != http.StatusOK {
			body, _ := readFirstChunk(resp)
			log.Printf("Attempt %d: API streaming request failed with status code %d: %s",
				attempt+1, resp.StatusCode, string(body))

			_ = drainAndCloseBody(resp.Body)
			continue
		}

		// Process the streaming response
		err = processStreamingResponse(resp, handler)

		// If we processed the stream successfully, return
		if err == nil {
			return nil
		}

		log.Printf("Attempt %d: error processing streaming response: %v", attempt+1, err)
		// If the error was due to the context being canceled, don't retry
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	return fmt.Errorf("streaming request to %s failed after %d attempts", req.URL, options.RetryAttempts+1)
}

// processStreamingResponse reads the response body line by line
// and calls the handler for each line
func processStreamingResponse(resp *http.Response, handler StreamChunkHandler) error {
	defer resp.Body.Close()

	// Use a scanner to read line by line
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		chunk := scanner.Bytes()

		// Skip empty lines
		if len(chunk) == 0 {
			continue
		}

		// Process the chunk
		if err := handler(chunk); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading streaming response: %w", err)
	}

	return nil
}

// readFirstChunk reads the first chunk of a response for error reporting
func readFirstChunk(resp *http.Response) ([]byte, error) {
	// Create a buffer to read the first 1KB
	buffer := make([]byte, 1024)
	n, err := resp.Body.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return buffer[:n], nil
}
