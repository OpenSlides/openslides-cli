package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/OpenSlides/openslides-cli/internal/logger"
)

const (
	httpScheme        = "http://"
	handleRequestPath = "/internal/handle_request"
	migrationsPath    = "/internal/migrations"
)

type Client struct {
	address  string
	password string
}

// New creates a new Client with the service address and password.
// Address should be in the format "host:port" (e.g., "localhost:9002").
func New(address, password string) *Client {
	logger.Debug("Creating new client for address: %s", address)
	return &Client{
		address:  address,
		password: password,
	}
}

// buildURL constructs the full URL from the client's address and the given path.
func (c *Client) buildURL(path string) string {
	return httpScheme + c.address + path
}

// SendAction sends an action request to the backend service.
// rawData should be a JSON array of action data objects.
func (c *Client) SendAction(action string, rawData []byte) (*http.Response, error) {
	logger.Info("Sending action: %s", action)
	logger.Debug("Action payload size: %d bytes", len(rawData))

	payload := []map[string]any{
		{
			"action": action,
			"data":   json.RawMessage(rawData),
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		logger.Error("Failed to marshal payload: %v", err)
		return nil, fmt.Errorf("marshalling payload: %w", err)
	}

	logger.Debug("Request body size: %d bytes", len(body))
	logger.Debug("Request body %s", body)

	url := c.buildURL(handleRequestPath)
	logger.Debug("Sending POST request to: %s", url)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		logger.Error("Failed to create request: %v", err)
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", base64.StdEncoding.EncodeToString([]byte(c.password)))

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	duration := time.Since(start)

	if err != nil {
		logger.Error("Request failed after %v: %v", duration, err)
		return nil, err
	}

	logger.Debug("Request completed in %v with status: %d", duration, resp.StatusCode)
	return resp, nil
}

// SendMigrations sends a migrations command to the backend
func (c *Client) SendMigrations(command string) (*http.Response, error) {
	logger.Info("Sending migrations command: %s", command)

	payload := map[string]string{
		"cmd": command,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		logger.Error("Failed to marshal migrations payload: %v", err)
		return nil, fmt.Errorf("marshalling payload: %w", err)
	}

	url := c.buildURL(migrationsPath)
	logger.Debug("Sending POST request to: %s", url)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		logger.Error("Failed to create request: %v", err)
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", base64.StdEncoding.EncodeToString([]byte(c.password)))

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	duration := time.Since(start)

	if err != nil {
		logger.Error("Migrations request failed after %v: %v", duration, err)
		return nil, err
	}

	logger.Debug("Migrations request completed in %v with status: %d", duration, resp.StatusCode)
	return resp, nil
}

// CheckResponse reads and checks the response, returning the body or an error.
func CheckResponse(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read response body: %v", err)
		return nil, fmt.Errorf("reading response: %w", err)
	}

	logger.Debug("Response body size: %d bytes", len(body))

	if resp.StatusCode != http.StatusOK {
		logger.Error("Request failed with status %d: %s", resp.StatusCode, string(body))
		return body, fmt.Errorf("request failed [%d]: %s", resp.StatusCode, string(body))
	}

	logger.Debug("Response successful")
	return body, nil
}
