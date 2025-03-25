package httputil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
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

var (
	httpClient *http.Client
	clientOnce sync.Once
)

// initClient initializes the HTTP client with default options
func initClient() {
	httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}
}

// SetClientOptions allows customization of the HTTP client
func SetClientOptions(options ClientOptions) {
	clientOnce.Do(func() {
		httpClient = &http.Client{
			Timeout: options.Timeout,
		}
	})
}

func drainAndCloseBody(body io.ReadCloser) error {
	_, err := io.Copy(io.Discard, body)
	if err != nil {
		return fmt.Errorf("error draining body: %w", err)
	}
	if err := body.Close(); err != nil {
		return fmt.Errorf("error closing body: %w", err)
	}
	return nil
}

func createRequest(ctx context.Context, details RequestDetails) (*http.Request, error) {
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

	return req, nil
}

func executeRequest(req *http.Request, options ClientOptions) ([]byte, error) {
	clientOnce.Do(initClient)

	var resp *http.Response
	var err error
	var body []byte

	for attempt := 0; attempt <= options.RetryAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(options.RetryDelay)
		}

		ctx, cancel := context.WithTimeout(req.Context(), options.Timeout)
		defer cancel()

		reqWithTimeout := req.WithContext(ctx)

		resp, err = httpClient.Do(reqWithTimeout)
		if err != nil {
			log.Printf("Attempt %d: error sending request to %s: %v", attempt+1, req.URL, err)
			continue
		}

		defer func() {
			if err := drainAndCloseBody(resp.Body); err != nil {
				log.Printf("Error closing response body: %v", err)
			}
		}()

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Attempt %d: error reading response from %s: %v", attempt+1, req.URL, err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return body, nil
		}

		log.Printf("Attempt %d: API request to %s failed with status code %d: %s", attempt+1, req.URL, resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("API request to %s failed after %d attempts", req.URL, options.RetryAttempts+1)
}

func SendRequest(ctx context.Context, details RequestDetails, options ClientOptions) ([]byte, error) {
	req, err := createRequest(ctx, details)
	if err != nil {
		return nil, err
	}

	return executeRequest(req, options)
}
