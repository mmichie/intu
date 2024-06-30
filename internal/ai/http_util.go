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

// Global HTTP client with timeouts
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func sendRequest(ctx context.Context, details RequestDetails) ([]byte, error) {
	jsonBody, err := json.Marshal(details.RequestBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", details.URL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
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
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer func() {
		// Ensure body is fully read and closed
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
