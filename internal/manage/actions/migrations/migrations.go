package migrations

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/OpenSlides/openslides-cli/internal/constants"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/OpenSlides/openslides-cli/internal/manage/client"
	"github.com/OpenSlides/openslides-cli/internal/utils"
)

const (
	MigrationsHelp      = "Wrapper to the OpenSlides backend migration tool"
	MigrationsHelpExtra = `Run database migrations on the OpenSlides datastore.

Examples:
  # Check migration status
  osmanage migrations stats \
    --address <myBackendManageIP>:9002 \
    --password-file my.instance.dir/secrets/internal_auth_password

  # Prepare migrations (dry run)
  osmanage migrations migrate \
    --address <myBackendManageIP>:9002 \
    --password-file my.instance.dir/secrets/internal_auth_password

  # Apply migrations to datastore
  osmanage migrations finalize \
    --address <myBackendManageIP>:9002 \
    --password-file my.instance.dir/secrets/internal_auth_password

  # Reset unapplied migrations
  osmanage migrations reset \
    --address <myBackendManageIP>:9002 \
    --password-file my.instance.dir/secrets/internal_auth_password

  # Check progress of running migration
  osmanage migrations progress \
    --address <myBackendManageIP>:9002 \
    --password-file my.instance.dir/secrets/internal_auth_password

  # Custom progress interval
  osmanage migrations finalize \
    --address <myBackendManageIP>:9002 \
    --password-file my.instance.dir/secrets/internal_auth_password \
    --interval 2s

Available commands:
  migrate                       Prepare migrations (dry run)
  finalize                      Apply migrations to datastore
  reset                         Reset unapplied migrations
  clear-collectionfield-tables  Clear auxiliary tables (offline only)
  stats                         Show migration statistics
  progress                      Check running migration progress`
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrations",
		Short: MigrationsHelp,
		Long:  MigrationsHelp + "\n\n" + MigrationsHelpExtra,
	}

	cmd.AddCommand(
		migrateCmd(),
		finalizeCmd(),
		resetCmd(),
		clearCollectionfieldTablesCmd(),
		statsCmd(),
		progressCmd(),
	)

	return cmd
}

func migrateCmd() *cobra.Command {
	return createMigrationCmd("migrate", "Prepare migrations but do not apply them to the datastore", true)
}

func finalizeCmd() *cobra.Command {
	return createMigrationCmd("finalize", "Prepare migrations and apply them to the datastore", true)
}

func resetCmd() *cobra.Command {
	return createMigrationCmd("reset", "Reset unapplied migrations", false)
}

func clearCollectionfieldTablesCmd() *cobra.Command {
	return createMigrationCmd("clear-collectionfield-tables", "Clear all data from auxiliary tables (only when OpenSlides is offline)", false)
}

func statsCmd() *cobra.Command {
	return createMigrationCmd("stats", "Print statistics about the current migration state", false)
}

func progressCmd() *cobra.Command {
	return createMigrationCmd("progress", "Query the progress of a currently running migration command", false)
}

// createMigrationCmd creates a migration subcommand with standard flags
func createMigrationCmd(name, description string, withProgressTracking bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name,
		Short: description,
		Args:  cobra.NoArgs,
	}

	address := cmd.Flags().StringP("address", "a", "", "address of the OpenSlides backendManage service (required)")
	passwordFile := cmd.Flags().String("password-file", "", "file with password for authorization (required)")

	_ = cmd.MarkFlagRequired("address")
	_ = cmd.MarkFlagRequired("password-file")

	var progressInterval *time.Duration
	if withProgressTracking {
		progressInterval = cmd.Flags().Duration("interval", constants.DefaultMigrationProgressInterval,
			"interval for progress checks (set 0 to disable progress tracking)")
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		logger.Info("=== MIGRATIONS: %s ===", strings.ToUpper(name))

		authPassword, err := utils.ReadPassword(*passwordFile)
		if err != nil {
			return fmt.Errorf("reading password: %w", err)
		}

		cl := client.New(*address, authPassword)

		// Execute migration command
		response, err := sendMigrationCommand(cl, name)
		if err != nil {
			return fmt.Errorf("executing migration command: %w", err)
		}

		// Handle response based on whether progress tracking is enabled
		if withProgressTracking && progressInterval != nil && *progressInterval > 0 && response.Running() {
			return trackMigrationProgress(cl, response, *progressInterval, name)
		}

		// No progress tracking - just print output
		output, err := response.GetOutput(name)
		if err != nil {
			return fmt.Errorf("formatting output: %w", err)
		}

		fmt.Print(output)
		return nil
	}

	return cmd
}

// sendMigrationCommand sends a migration command with retry logic
func sendMigrationCommand(cl *client.Client, command string) (*MigrationResponse, error) {
	logger.Debug("Sending migration command: %s", command)

	ctx, cancel := context.WithTimeout(context.Background(), constants.MigrationTotalTimeout)
	defer cancel()

	var lastErr error

	for attempt := 0; attempt < constants.MigrationMaxRetries; attempt++ {
		// Check if context expired
		if ctx.Err() != nil {
			return nil, fmt.Errorf("migration command timed out after %v: %w", constants.MigrationTotalTimeout, ctx.Err())
		}

		// Wait before retry (except first attempt)
		if attempt > 0 {
			logger.Warn("Retry attempt %d/%d after %v (previous error: %v)",
				attempt, constants.MigrationMaxRetries, constants.MigrationRetryDelay, lastErr)

			select {
			case <-time.After(constants.MigrationRetryDelay):
				// Continue to next attempt
			case <-ctx.Done():
				return nil, fmt.Errorf("migration command cancelled during retry: %w", ctx.Err())
			}
		}

		// Send request
		resp, err := cl.SendMigrations(command)
		if err != nil {
			lastErr = fmt.Errorf("sending request: %w", err)
			if isRetryableError(err) && attempt < constants.MigrationMaxRetries-1 {
				logger.Debug("Retryable error: %v", err)
				continue
			}
			return nil, lastErr
		}

		// Check response
		body, err := client.CheckResponse(resp)
		if err != nil {
			lastErr = err
			if isRetryableError(err) && attempt < constants.MigrationMaxRetries-1 {
				logger.Debug("Retryable error: %v", err)
				continue
			}
			return nil, lastErr
		}

		// Parse response
		var migrationResp MigrationResponse
		if err := json.Unmarshal(body, &migrationResp); err != nil {
			logger.Error("Failed to unmarshal migration response: %v", err)
			return nil, fmt.Errorf("unmarshalling response: %w", err)
		}

		logger.Debug("Migration response - Success: %v, Status: %s, Running: %v",
			migrationResp.Success, migrationResp.Status, migrationResp.Running())

		return &migrationResp, nil
	}

	return nil, fmt.Errorf("migration command failed after %d retries: %w", constants.MigrationMaxRetries, lastErr)
}

// trackMigrationProgress polls migration progress until completion
func trackMigrationProgress(cl *client.Client, initialResponse *MigrationResponse, interval time.Duration, command string) error {
	fmt.Println("Progress:")
	logger.Debug("Starting progress tracking with interval: %v", interval)

	for {
		time.Sleep(interval)

		response, err := sendMigrationCommand(cl, "progress")
		if err != nil {
			return fmt.Errorf("checking progress: %w", err)
		}

		// Print progress output
		output, err := response.GetOutput("progress")
		if err != nil {
			return fmt.Errorf("formatting progress output: %w", err)
		}
		fmt.Print(output)

		// Check if migration failed
		if response.Faulty() {
			logger.Error("Migration command failed")
			return fmt.Errorf("migration failed: %s", response.Exception)
		}

		// Check if migration completed
		if !response.Running() {
			logger.Info("Migration completed")
			break
		}
	}

	return nil
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Network-related errors
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporary failure",
		"no such host",
		"network is unreachable",
		"eof",
		"broken pipe",
		"i/o timeout",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	// HTTP server errors (5xx)
	serverErrors := []string{"server error", "503", "502", "504"}
	for _, code := range serverErrors {
		if strings.Contains(errStr, code) {
			return true
		}
	}

	return false
}

// MigrationResponse represents the response from a migration command
type MigrationResponse struct {
	Success   bool            `json:"success"`
	Status    string          `json:"status"`
	Output    string          `json:"output"`
	Exception string          `json:"exception"`
	Stats     json.RawMessage `json:"stats"`
}

// GetOutput returns the formatted output for the migration response
func (mr *MigrationResponse) GetOutput(command string) (string, error) {
	if mr.Faulty() {
		return mr.formatAll()
	}
	if command == "stats" {
		return mr.formatStats()
	}
	return mr.Output, nil
}

// formatStats formats the stats JSON into a readable string
func (mr *MigrationResponse) formatStats() (string, error) {
	var stats map[string]any
	if err := json.Unmarshal(mr.Stats, &stats); err != nil {
		return "", fmt.Errorf("unmarshalling stats: %w", err)
	}

	var sb strings.Builder
	for _, field := range constants.MigrationStatsFields {
		if value, ok := stats[field]; ok {
			sb.WriteString(fmt.Sprintf("%s: %v\n", field, value))
		}
	}

	return sb.String(), nil
}

// formatAll formats all response fields
func (mr *MigrationResponse) formatAll() (string, error) {
	return fmt.Sprintf("Success: %v\nStatus: %s\nOutput: %s\nException: %s\n",
		mr.Success, mr.Status, mr.Output, mr.Exception), nil
}

// Faulty returns true if the migration failed
func (mr *MigrationResponse) Faulty() bool {
	return !mr.Success || mr.Exception != ""
}

// Running returns true if the migration is currently in progress
func (mr *MigrationResponse) Running() bool {
	return mr.Status == constants.MigrationStatusRunning
}
