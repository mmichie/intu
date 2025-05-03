package httputil

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
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

		// Create a goroutine to cancel the context after a hard deadline
		// This is a failsafe in case the normal cancelation doesn't work
		hardTimeoutDone := make(chan struct{})
		go func() {
			select {
			case <-time.After(options.Timeout + 5*time.Second):
				// Hard timeout - force cancel
				log.Printf("Hard timeout triggered for streaming request after %v", options.Timeout+5*time.Second)
				cancel()
			case <-hardTimeoutDone:
				// Normal completion
				return
			}
		}()

		reqWithTimeout := req.WithContext(attemptCtx)

		// Send the request
		resp, err := httpClient.Do(reqWithTimeout)
		if err != nil {
			cancel()
			close(hardTimeoutDone)
			log.Printf("Attempt %d: error sending streaming request: %v", attempt+1, err)
			continue
		}

		// Check if the response is successful
		if resp.StatusCode != http.StatusOK {
			body, _ := readFirstChunk(resp)
			log.Printf("Attempt %d: API streaming request failed with status code %d: %s",
				attempt+1, resp.StatusCode, string(body))

			_ = drainAndCloseBody(resp.Body)
			cancel()
			close(hardTimeoutDone)
			continue
		}

		// Process the streaming response with additional timeout protection
		processDone := make(chan error, 1)

		go func() {
			processDone <- processStreamingResponse(resp, handler)
		}()

		// Wait for processing to complete or timeout
		var processingErr error
		select {
		case err := <-processDone:
			processingErr = err
		case <-time.After(options.Timeout + 3*time.Second):
			processingErr = fmt.Errorf("processing streaming response timed out after %v", options.Timeout+3*time.Second)
			// Try to close the response body
			_ = resp.Body.Close()
		}

		// Clean up
		cancel()
		close(hardTimeoutDone)

		// If we processed the stream successfully, return
		if processingErr == nil {
			return nil
		}

		log.Printf("Attempt %d: error processing streaming response: %v", attempt+1, processingErr)
		// If the error was due to the context being canceled, don't retry
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// For timeout errors, return immediately rather than retrying
		if strings.Contains(processingErr.Error(), "timeout") ||
			strings.Contains(processingErr.Error(), "timed out") {
			return processingErr
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

	lineCount := 0
	for scanner.Scan() {
		lineCount++
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

// SendTextStreamingRequest sends a streaming request and processes chunks as text
// This is a convenience wrapper around SendStreamingRequest that converts byte chunks to text
func SendTextStreamingRequest(
	ctx context.Context,
	details RequestDetails,
	options ClientOptions,
	handler TextStreamHandler,
) error {
	// Create a wrapper that converts bytes to text
	byteHandler := func(chunk []byte) error {
		return handler(string(chunk))
	}

	// Use the byte-based streaming function
	return SendStreamingRequest(ctx, details, options, byteHandler)
}
