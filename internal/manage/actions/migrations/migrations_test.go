package migrations

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/OpenSlides/openslides-cli/internal/constants"
)

func TestMigrationResponse_Faulty(t *testing.T) {
	tests := []struct {
		name      string
		resp      MigrationResponse
		wantFault bool
	}{
		{
			"success no exception",
			MigrationResponse{Success: true, Exception: ""},
			false,
		},
		{
			"failure",
			MigrationResponse{Success: false, Exception: ""},
			true,
		},
		{
			"success with exception",
			MigrationResponse{Success: true, Exception: "error occurred"},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.resp.Faulty(); got != tt.wantFault {
				t.Errorf("Faulty() = %v, want %v", got, tt.wantFault)
			}
		})
	}
}

func TestMigrationResponse_Running(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		running bool
	}{
		{"running", constants.MigrationStatusRunning, true},
		{"completed", "completed", false},
		{"failed", "failed", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := MigrationResponse{Status: tt.status}
			if got := resp.Running(); got != tt.running {
				t.Errorf("Running() = %v, want %v", got, tt.running)
			}
		})
	}
}

func TestMigrationResponse_GetOutput(t *testing.T) {
	t.Run("normal output", func(t *testing.T) {
		resp := MigrationResponse{
			Success: true,
			Output:  "Migration completed",
		}
		output, err := resp.GetOutput("migrate")
		if err != nil {
			t.Errorf("GetOutput() error = %v", err)
		}
		if output != "Migration completed" {
			t.Errorf("Expected 'Migration completed', got %s", output)
		}
	})

	t.Run("stats command", func(t *testing.T) {
		stats := map[string]any{
			"current_migration_index":      68,
			"target_migration_index":       70,
			"positions":                    6,
			"events":                       70,
			"partially_migrated_positions": 0,
			"fully_migrated_positions":     0,
			"status":                       "finalization_required",
		}
		statsJSON, _ := json.Marshal(stats)
		resp := MigrationResponse{
			Success: true,
			Stats:   statsJSON,
		}
		output, err := resp.GetOutput("stats")
		if err != nil {
			t.Errorf("GetOutput() error = %v", err)
		}

		// Verify all expected fields are present
		// Using subset of MigrationStatsFields for validation
		expectedFields := []string{
			"current_migration_index",
			"target_migration_index",
			"positions",
			"events",
			"status",
		}
		for _, field := range expectedFields {
			if !strings.Contains(output, field) {
				t.Errorf("Expected %s in stats output", field)
			}
		}
	})

	t.Run("faulty response", func(t *testing.T) {
		resp := MigrationResponse{
			Success:   false,
			Exception: "Migration failed",
		}
		output, err := resp.GetOutput("migrate")
		if err != nil {
			t.Errorf("GetOutput() error = %v", err)
		}
		if !strings.Contains(output, "Migration failed") {
			t.Error("Expected exception in output")
		}
	})
}

func TestMigrationResponse_FormatStats(t *testing.T) {
	t.Run("ordered output", func(t *testing.T) {
		stats := map[string]any{
			"status":                       "finalization_required",
			"events":                       70,
			"current_migration_index":      68,
			"target_migration_index":       70,
			"positions":                    6,
			"partially_migrated_positions": 0,
			"fully_migrated_positions":     0,
		}
		statsJSON, _ := json.Marshal(stats)
		resp := &MigrationResponse{Stats: statsJSON}

		output, err := resp.formatStats()
		if err != nil {
			t.Errorf("formatStats() error = %v", err)
		}

		// Verify order: current_migration_index should come before status
		lines := strings.Split(output, "\n")
		var currentIdx, statusIdx int
		for i, line := range lines {
			if strings.HasPrefix(line, "current_migration_index:") {
				currentIdx = i
			}
			if strings.HasPrefix(line, "status:") {
				statusIdx = i
			}
		}

		if currentIdx >= statusIdx {
			t.Error("Expected current_migration_index to appear before status in output")
		}

		// Verify all fields are present
		if !strings.Contains(output, "current_migration_index: 68") {
			t.Error("Expected current_migration_index: 68")
		}
		if !strings.Contains(output, "status: finalization_required") {
			t.Error("Expected status: finalization_required")
		}
	})

	t.Run("missing optional fields", func(t *testing.T) {
		stats := map[string]any{
			"status":                  "no_migration_required",
			"current_migration_index": 70,
		}
		statsJSON, _ := json.Marshal(stats)
		resp := &MigrationResponse{Stats: statsJSON}

		output, err := resp.formatStats()
		if err != nil {
			t.Errorf("formatStats() error = %v", err)
		}

		// Should still include present fields
		if !strings.Contains(output, "status:") {
			t.Error("Expected status in output")
		}
		if !strings.Contains(output, "current_migration_index:") {
			t.Error("Expected current_migration_index in output")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		resp := &MigrationResponse{Stats: json.RawMessage("invalid json")}

		_, err := resp.formatStats()
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		errMsg    string
		retryable bool
	}{
		{"nil error", "", false},
		{"connection refused", "connection refused", true},
		{"connection reset", "connection reset by peer", true},
		{"timeout", "i/o timeout", true},
		{"eof", "unexpected EOF", true},
		{"server error 503", "server returned 503", true},
		{"server error 502", "bad gateway 502", true},
		{"server error 504", "gateway timeout 504", true},
		{"client error 404", "404 not found", false},
		{"auth error", "unauthorized", false},
		{"parse error", "invalid JSON", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.errMsg != "" {
				err = &testError{msg: tt.errMsg}
			}

			if got := isRetryableError(err); got != tt.retryable {
				t.Errorf("isRetryableError() = %v, want %v for error: %s", got, tt.retryable, tt.errMsg)
			}
		})
	}
}

// Helper type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
