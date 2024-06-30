package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RequestDetails holds the details for an HTTP request
type RequestDetails struct {
	URL               string
	APIKey            string
	RequestBody       interface{}
	AdditionalHeaders map[string]string
}

// ClientOptions holds options for customizing the HTTP client
type ClientOptions struct {
	Timeout       time.Duration
	RetryAttempts int
	RetryDelay    time.Duration
}

// Global HTTP client with default options
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// SetClientOptions allows customization of the HTTP client
func SetClientOptions(options ClientOptions) {
	httpClient = &http.Client{
		Timeout: options.Timeout,
	}
	// Add retry logic here if needed
}

func drainAndCloseBody(body io.ReadCloser) {
	_, _ = io.Copy(io.Discard, body)
	_ = body.Close()
}

func sendRequest(ctx context.Context, details RequestDetails) ([]byte, error) {
	jsonBody, err := json.Marshal(details.RequestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", details.URL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request for URL %s: %w", details.URL, err)
	}

	req.Header.Set("Content-Type", "application/json")
	if details.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+details.APIKey)
	}

	for key, value := range details.AdditionalHeaders {
		req.Header.Set(key, value)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request to %s: %w", details.URL, err)
	}
	defer drainAndCloseBody(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response from %s: %w", details.URL, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request to %s failed with status code %d: %s", details.URL, resp.StatusCode, string(body))
	}

	return body, nil
}
