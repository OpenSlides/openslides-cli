package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OpenSlides/openslides-cli/internal/constants"
)

func TestNew(t *testing.T) {
	address := "localhost:9002"
	password := "test-password"

	client := New(address, password)

	if client.address != address {
		t.Errorf("Expected address %s, got %s", address, client.address)
	}
	if client.password != password {
		t.Errorf("Expected password %s, got %s", password, client.password)
	}
}

func TestBuildURL(t *testing.T) {
	client := New("localhost:9002", "password")

	tests := []struct {
		name string
		path string
		want string
	}{
		{"handle request", constants.BackendHandleRequestPath, "http://localhost:9002/internal/handle_request"},
		{"migrations", constants.BackendMigrationsPath, "http://localhost:9002/internal/migrations"},
		{"custom path", "/custom", "http://localhost:9002/custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.buildURL(tt.path)
			if got != tt.want {
				t.Errorf("buildURL(%s) = %s, want %s", tt.path, got, tt.want)
			}
		})
	}
}

func TestSendAction(t *testing.T) {
	t.Run("successful request", func(t *testing.T) {
		var receivedAuth string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != constants.BackendHandleRequestPath {
				t.Errorf("Expected path %s, got %s", constants.BackendHandleRequestPath, r.URL.Path)
			}
			if r.Method != "POST" {
				t.Errorf("Expected POST, got %s", r.Method)
			}
			if r.Header.Get("Content-Type") != constants.BackendContentType {
				t.Errorf("Expected Content-Type: %s", constants.BackendContentType)
			}

			receivedAuth = r.Header.Get("Authorization")
			if receivedAuth == "" {
				t.Error("Expected Authorization header")
			}

			// Verify request body structure
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"action"`) {
				t.Error("Expected 'action' field in request body")
			}
			if !strings.Contains(string(body), `"data"`) {
				t.Error("Expected 'data' field in request body")
			}

			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"success":true}`)); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))
		defer server.Close()

		address := strings.TrimPrefix(server.URL, constants.BackendHTTPScheme)
		cl := New(address, "test-password")

		resp, err := cl.SendAction("test.action", []byte(`[{"id":1}]`))
		if err != nil {
			t.Fatalf("SendAction() error = %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		if receivedAuth == "" {
			t.Error("Authorization header was not received by server")
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			if _, err := w.Write([]byte(`{"error":"server error"}`)); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))
		defer server.Close()

		address := strings.TrimPrefix(server.URL, constants.BackendHTTPScheme)
		cl := New(address, "test-password")

		resp, err := cl.SendAction("test.action", []byte(`[{"id":1}]`))
		if err != nil {
			t.Fatalf("SendAction() error = %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", resp.StatusCode)
		}
	})

	t.Run("invalid json data", func(t *testing.T) {
		cl := New("localhost:9002", "test-password")

		// SendAction wraps rawData as json.RawMessage, which validates JSON
		// Invalid JSON should cause an error during marshalling
		resp, err := cl.SendAction("test.action", []byte(`invalid json`))
		if err == nil {
			t.Error("Expected error for invalid JSON data")
			if resp != nil {
				_ = resp.Body.Close()
			}
			return
		}

		// Verify it's a marshalling error
		if !strings.Contains(err.Error(), "marshalling payload") {
			t.Errorf("Expected marshalling error, got: %v", err)
		}

		// resp should be nil when there's an error before sending
		if resp != nil {
			t.Error("Expected nil response when marshalling fails")
			_ = resp.Body.Close()
		}
	})
}

func TestSendMigrations(t *testing.T) {
	t.Run("successful migrations request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != constants.BackendMigrationsPath {
				t.Errorf("Expected path %s, got %s", constants.BackendMigrationsPath, r.URL.Path)
			}
			if r.Method != "POST" {
				t.Errorf("Expected POST, got %s", r.Method)
			}

			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"cmd"`) {
				t.Error("Expected cmd field in request body")
			}
			if !strings.Contains(string(body), `"stats"`) {
				t.Error("Expected 'stats' command in request body")
			}

			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"success":true,"status":"completed"}`)); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))
		defer server.Close()

		address := strings.TrimPrefix(server.URL, constants.BackendHTTPScheme)
		cl := New(address, "test-password")

		resp, err := cl.SendMigrations("stats")
		if err != nil {
			t.Errorf("SendMigrations() error = %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("migrations error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			if _, err := w.Write([]byte(`{"error":"invalid command"}`)); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))
		defer server.Close()

		address := strings.TrimPrefix(server.URL, constants.BackendHTTPScheme)
		cl := New(address, "test-password")

		resp, err := cl.SendMigrations("invalid")
		if err != nil {
			t.Errorf("SendMigrations() error = %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", resp.StatusCode)
		}
	})
}

func TestCheckResponse(t *testing.T) {
	t.Run("success response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"success":true}`)); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))
		defer server.Close()

		resp, _ := http.Get(server.URL)
		body, err := CheckResponse(resp)
		if err != nil {
			t.Errorf("CheckResponse() error = %v", err)
		}
		if !strings.Contains(string(body), "success") {
			t.Error("Expected success in response body")
		}
	})

	t.Run("error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			if _, err := w.Write([]byte(`{"error":"bad request"}`)); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		}))
		defer server.Close()

		resp, _ := http.Get(server.URL)
		body, err := CheckResponse(resp)
		if err == nil {
			t.Error("Expected error for non-200 response")
		}
		if !strings.Contains(err.Error(), "400") {
			t.Errorf("Expected 400 in error message, got: %v", err)
		}
		// Body should still be returned even on error
		if len(body) == 0 {
			t.Error("Expected body to be returned even on error")
		}
	})

	t.Run("empty response body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			// No body
		}))
		defer server.Close()

		resp, _ := http.Get(server.URL)
		body, err := CheckResponse(resp)
		if err != nil {
			t.Errorf("CheckResponse() error = %v", err)
		}
		if len(body) != 0 {
			t.Errorf("Expected empty body, got %d bytes", len(body))
		}
	})

	t.Run("various status codes", func(t *testing.T) {
		tests := []struct {
			name       string
			statusCode int
			wantErr    bool
		}{
			{"200 OK", http.StatusOK, false},
			{"201 Created", http.StatusCreated, true}, // Not StatusOK
			{"400 Bad Request", http.StatusBadRequest, true},
			{"401 Unauthorized", http.StatusUnauthorized, true},
			{"404 Not Found", http.StatusNotFound, true},
			{"500 Internal Server Error", http.StatusInternalServerError, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.statusCode)
					if _, err := w.Write([]byte(`{"status":"test"}`)); err != nil {
						t.Fatalf("failed to write response: %v", err)
					}
				}))
				defer server.Close()

				resp, _ := http.Get(server.URL)
				_, err := CheckResponse(resp)

				if (err != nil) != tt.wantErr {
					t.Errorf("CheckResponse() error = %v, wantErr %v", err, tt.wantErr)
				}
			})
		}
	})
}
