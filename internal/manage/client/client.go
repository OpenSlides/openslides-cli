package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/OpenSlides/openslides-cli/internal/constants"
	"github.com/OpenSlides/openslides-cli/internal/logger"
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
	return constants.BackendHTTPScheme + c.address + path
}

// escapeForShell escapes single quotes in a string for safe use in shell commands.
func escapeForShell(s string) string {
	return strings.ReplaceAll(s, "'", "'\"'\"'")
}

// logCurlCommand logs a curl command that can be used to reproduce the request.
func logCurlCommand(method, url string, headers map[string]string, body []byte) {
	var parts []string
	parts = append(parts, fmt.Sprintf("curl -X %s '%s'", method, url))

	for key, value := range headers {
		parts = append(parts, fmt.Sprintf("-H '%s: %s'", key, value))
	}

	if len(body) > 0 {
		parts = append(parts, fmt.Sprintf("-d '%s'", escapeForShell(string(body))))
	}

	logger.Debug("Equivalent curl command:\n  %s", strings.Join(parts, " \\\n  "))
}

// logResponseDetails logs response headers and metadata.
func logResponseDetails(resp *http.Response, duration time.Duration) {
	logger.Debug("Response status: %d %s", resp.StatusCode, resp.Status)
	logger.Debug("Response headers:")
	for key, values := range resp.Header {
		for _, value := range values {
			logger.Debug("  %s: %s", key, value)
		}
	}
	logger.Debug("Request completed in %v", duration)
}

// SendAction sends an action request to the backend service.
// rawData should be a JSON array of action data objects.
func (c *Client) SendAction(action string, rawData []byte) (*http.Response, error) {
	logger.Info("Sending action: %s", action)

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

	url := c.buildURL(constants.BackendHandleRequestPath)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		logger.Error("Failed to create request: %v", err)
		return nil, fmt.Errorf("creating request: %w", err)
	}

	authHeader := base64.StdEncoding.EncodeToString([]byte(c.password))
	req.Header.Set("Content-Type", constants.BackendContentType)
	req.Header.Set("Authorization", authHeader)

	logCurlCommand("POST", url, map[string]string{
		"Content-Type":  constants.BackendContentType,
		"Authorization": authHeader,
	}, body)

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	duration := time.Since(start)

	if err != nil {
		logger.Error("Request failed after %v: %v", duration, err)
		return nil, err
	}

	logResponseDetails(resp, duration)

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

	url := c.buildURL(constants.BackendMigrationsPath)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		logger.Error("Failed to create request: %v", err)
		return nil, fmt.Errorf("creating request: %w", err)
	}

	authHeader := base64.StdEncoding.EncodeToString([]byte(c.password))
	req.Header.Set("Content-Type", constants.BackendContentType)
	req.Header.Set("Authorization", authHeader)

	logCurlCommand("POST", url, map[string]string{
		"Content-Type":  constants.BackendContentType,
		"Authorization": authHeader,
	}, body)

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	duration := time.Since(start)

	if err != nil {
		logger.Error("Migrations request failed after %v: %v", duration, err)
		return nil, err
	}

	logResponseDetails(resp, duration)

	return resp, nil
}

// CheckResponse reads and checks the response, returning the body or an error.
func CheckResponse(resp *http.Response) ([]byte, error) {
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read response body: %v", err)
		return nil, fmt.Errorf("reading response: %w", err)
	}

	logger.Debug("Response body (%d bytes): %s", len(body), string(body))

	if resp.StatusCode != http.StatusOK {
		logger.Error("Request failed with status %d: %s", resp.StatusCode, string(body))
		return body, fmt.Errorf("request failed [%d]: %s", resp.StatusCode, string(body))
	}

	logger.Debug("Response successful")
	return body, nil
}
