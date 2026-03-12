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
	pb "github.com/OpenSlides/openslides-cli/proto/osmanage"
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

		response, err := ExecuteMigrationCommand(cl, name)
		if err != nil {
			return fmt.Errorf("executing migration command: %w", err)
		}

		if withProgressTracking && progressInterval != nil && *progressInterval > 0 && (Running(response) || Finalizing(response)) {
			fmt.Println("Progress:")

			var stopCondition func(*pb.MigrationsResponse) bool
			if name == "finalize" {
				stopCondition = func(r *pb.MigrationsResponse) bool { return !Running(r) && !Finalizing(r) }
			} else {
				stopCondition = func(r *pb.MigrationsResponse) bool { return !Running(r) }
			}

			printCallback := func(update *pb.MigrationsProgressResponse) error {
				fmt.Print(update.Output)
				return nil
			}

			return TrackMigrationProgress(cl, *progressInterval, stopCondition, printCallback)
		}

		output, err := GetOutput(response, name)
		if err != nil {
			return fmt.Errorf("formatting output: %w", err)
		}

		fmt.Print(output)
		return nil
	}

	return cmd
}

// Internal type for HTTP response (stats is JSON object)
type migrationsHTTPResponse struct {
	Success   bool            `json:"success"`
	Status    string          `json:"status"`
	Output    string          `json:"output"`
	Exception string          `json:"exception"`
	Stats     json.RawMessage `json:"stats"`
}

// ExecuteMigrationCommand sends a migration command to the backend with retry logic.
func ExecuteMigrationCommand(cl *client.Client, command string) (*pb.MigrationsResponse, error) {
	logger.Debug("Executing migration command: %s", command)

	ctx, cancel := context.WithTimeout(context.Background(), constants.MigrationTotalTimeout)
	defer cancel()

	var lastErr error

	for attempt := range constants.MigrationMaxRetries {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("migration command timed out after %v: %w", constants.MigrationTotalTimeout, ctx.Err())
		}

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

		resp, err := cl.SendMigrations(command)
		if err != nil {
			lastErr = fmt.Errorf("sending request: %w", err)
			if isRetryableError(err) && attempt < constants.MigrationMaxRetries-1 {
				logger.Debug("Retryable error: %v", err)
				continue
			}
			return nil, lastErr
		}

		body, err := client.CheckResponse(resp)
		if err != nil {
			lastErr = err
			if isRetryableError(err) && attempt < constants.MigrationMaxRetries-1 {
				logger.Debug("Retryable error: %v", err)
				continue
			}
			return nil, lastErr
		}

		var httpResp migrationsHTTPResponse
		if err := json.Unmarshal(body, &httpResp); err != nil {
			logger.Error("Failed to unmarshal migration response: %v", err)
			return nil, fmt.Errorf("unmarshalling response: %w", err)
		}

		migrationResp := &pb.MigrationsResponse{
			Success:   httpResp.Success,
			Status:    httpResp.Status,
			Output:    httpResp.Output,
			Exception: httpResp.Exception,
			Stats:     string(httpResp.Stats),
		}

		logger.Debug("Migration response - Success: %v, Status: %s, Running: %v, Finalizing: %v",
			migrationResp.Success, migrationResp.Status, Running(migrationResp), Finalizing(migrationResp))

		return migrationResp, nil
	}

	return nil, fmt.Errorf("migration command failed after %d retries: %w", constants.MigrationMaxRetries, lastErr)
}

// TrackMigrationProgress polls migration progress and sends updates to the callback.
func TrackMigrationProgress(
	cl *client.Client,
	interval time.Duration,
	stopCondition func(*pb.MigrationsResponse) bool,
	callback func(*pb.MigrationsProgressResponse) error,
) error {
	logger.Debug("Starting progress tracking with interval: %v", interval)

	for {
		time.Sleep(interval)

		response, err := ExecuteMigrationCommand(cl, "progress")
		if err != nil {
			return fmt.Errorf("checking progress: %w", err)
		}

		update := &pb.MigrationsProgressResponse{
			Output:    response.Output,
			Running:   Running(response) || Finalizing(response),
			Success:   response.Success,
			Exception: response.Exception,
		}

		if err := callback(update); err != nil {
			return fmt.Errorf("progress callback error: %w", err)
		}

		if Faulty(response) {
			logger.Error("Migration command failed")
			return fmt.Errorf("migration failed: %s", response.Exception)
		}

		if stopCondition(response) {
			logger.Info("Migration completed")
			break
		}
	}

	return nil
}

// GetOutput returns the formatted output for the migration response
func GetOutput(mr *pb.MigrationsResponse, command string) (string, error) {
	if Faulty(mr) {
		return formatAll(mr)
	}
	if command == "stats" {
		return FormatStats(mr.Stats)
	}
	return mr.Output, nil
}

// FormatStats formats the stats bytes into a readable string (exported for gRPC use)
func FormatStats(stats string) (string, error) {
	if stats == "" {
		return "", nil
	}

	var statsMap map[string]any
	if err := json.Unmarshal([]byte(stats), &statsMap); err != nil {
		return "", fmt.Errorf("unmarshalling stats: %w", err)
	}

	var sb strings.Builder
	for _, field := range constants.MigrationStatsFields {
		if value, ok := statsMap[field]; ok {
			fmt.Fprintf(&sb, "%s: %v\n", field, value)
		}
	}

	return sb.String(), nil
}

// formatAll formats all response fields
func formatAll(mr *pb.MigrationsResponse) (string, error) {
	return fmt.Sprintf("Success: %v\nStatus: %s\nOutput: %s\nException: %s\n",
		mr.Success, mr.Status, mr.Output, mr.Exception), nil
}

// Faulty returns true if the migration failed
func Faulty(mr *pb.MigrationsResponse) bool {
	return !mr.Success || mr.Exception != "" ||
		mr.Status == constants.MigrationStatusFailed ||
		mr.Status == constants.FinalizationStatusFailed
}

// Running returns true if the migration is currently in progress
func Running(mr *pb.MigrationsResponse) bool {
	return mr.Status == constants.MigrationStatusRunning
}

// Finalizing returns true if the migration finalization is currently in progress
func Finalizing(mr *pb.MigrationsResponse) bool {
	return mr.Status == constants.FinalizationStatusRunning
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
