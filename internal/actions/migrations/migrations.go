package migrations

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/OpenSlides/openslides-cli/internal/client"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/OpenSlides/openslides-cli/internal/utils"

	"github.com/spf13/cobra"
)

const (
	MigrationsHelp      = "Wrapper to the OpenSlides backend migration tool"
	MigrationsHelpExtra = `Run database migrations on the OpenSlides datastore.
See help text for the respective commands for more information.`

	defaultInterval  = 1 * time.Second
	migrationRunning = "migration_running"
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
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Prepare migrations but do not apply them to the datastore",
		Args:  cobra.NoArgs,
	}
	return setupMigrationCmd(cmd, true)
}

func finalizeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "finalize",
		Short: "Prepare migrations and apply them to the datastore",
		Args:  cobra.NoArgs,
	}
	return setupMigrationCmd(cmd, true)
}

func resetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset unapplied migrations",
		Args:  cobra.NoArgs,
	}
	return setupMigrationCmd(cmd, false)
}

func clearCollectionfieldTablesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear-collectionfield-tables",
		Short: "Clear all data from auxiliary tables (only when OpenSlides is offline)",
		Args:  cobra.NoArgs,
	}
	return setupMigrationCmd(cmd, false)
}

func statsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Print statistics about the current migration state",
		Args:  cobra.NoArgs,
	}
	return setupMigrationCmd(cmd, false)
}

func progressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "progress",
		Short: "Query the progress of a currently running migration command",
		Args:  cobra.NoArgs,
	}
	return setupMigrationCmd(cmd, false)
}

func setupMigrationCmd(cmd *cobra.Command, withInterval bool) *cobra.Command {
	address := cmd.Flags().StringP("address", "a", "localhost:9002", "address of the OpenSlides backendManage service")
	passwordFile := cmd.Flags().String("password-file", "secrets/internal_auth_password", "file with password for authorization")

	var interval *time.Duration
	if withInterval {
		interval = cmd.Flags().Duration("interval", defaultInterval,
			"interval of progress calls on running migrations, set 0 to disable progress")
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		logger.Info("=== MIGRATIONS: %s ===", strings.ToUpper(cmd.Use))

		authPassword, err := utils.ReadPassword(*passwordFile)
		if err != nil {
			return fmt.Errorf("reading password: %w", err)
		}

		cl := client.New(*address, authPassword)

		return runMigrations(cl, cmd.Use, interval)
	}

	return cmd
}

func runMigrations(cl *client.Client, command string, intervalFlag *time.Duration) error {
	logger.Debug("Running migrations command: %s", command)

	mR, err := executeMigrationsCommand(cl, command)
	if err != nil {
		return fmt.Errorf("executing migrations command: %w", err)
	}

	var interval time.Duration
	if intervalFlag != nil {
		interval = *intervalFlag
	}

	// If no interval or not running, just print and return
	if interval == 0 || !mR.Running() {
		output, err := mR.GetOutput(command)
		if err != nil {
			return fmt.Errorf("parsing migrations response: %w", err)
		}
		fmt.Print(output)
		return nil
	}

	// Track progress with intervals
	fmt.Println("Progress:")
	logger.Debug("Starting progress tracking with interval: %v", interval)

	for {
		time.Sleep(interval)

		mR, err := executeMigrationsCommand(cl, "progress")
		if err != nil {
			return fmt.Errorf("checking progress: %w", err)
		}

		if mR.Faulty() {
			logger.Error("Migration command failed")
			out, err := mR.GetOutput("progress")
			if err != nil {
				return fmt.Errorf("parsing error response: %w", err)
			}
			fmt.Print(out)
		} else {
			out, err := mR.GetOutput("progress")
			if err != nil {
				return fmt.Errorf("error parsing progress output: %w", err)
			}
			fmt.Print(out)
		}

		if !mR.Running() {
			logger.Info("Migration completed")
			break
		}
	}

	return nil
}

func executeMigrationsCommand(cl *client.Client, command string) (MigrationResponse, error) {
	logger.Debug("Executing migrations command: %s", command)

	const maxRetries = 5
	const retryDelay = 5 * time.Second
	const totalTimeout = 3 * time.Minute // Max time for all retries combined

	ctx, cancel := context.WithTimeout(context.Background(), totalTimeout)
	defer cancel()

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Check if context expired
		if ctx.Err() != nil {
			return MigrationResponse{}, fmt.Errorf("migrations command timed out after %v: %w", totalTimeout, ctx.Err())
		}

		if attempt > 0 {
			logger.Warn("Retry attempt %d/%d after %v (previous error: %v)",
				attempt, maxRetries, retryDelay, lastErr)

			// Sleep with context awareness
			select {
			case <-time.After(retryDelay):
				// Continue to next attempt
			case <-ctx.Done():
				return MigrationResponse{}, fmt.Errorf("migrations command cancelled during retry: %w", ctx.Err())
			}
		}

		resp, err := cl.SendMigrations(command)
		if err != nil {
			lastErr = fmt.Errorf("sending migrations request: %w", err)
			if isRetryableError(err) && attempt < maxRetries-1 {
				logger.Debug("Retryable error: %v", err)
				continue
			}
			return MigrationResponse{}, lastErr
		}

		body, err := client.CheckResponse(resp)
		if err != nil {
			lastErr = err
			if isRetryableError(err) && attempt < maxRetries-1 {
				logger.Debug("Retryable error: %v", err)
				continue
			}
			return MigrationResponse{}, lastErr
		}

		var mR MigrationResponse
		if err := json.Unmarshal(body, &mR); err != nil {
			logger.Error("Failed to unmarshal migrations response: %v", err)
			return MigrationResponse{}, fmt.Errorf("unmarshalling migration response: %w", err)
		}

		logger.Debug("Migration response - Success: %v, Status: %s, Running: %v",
			mR.Success, mR.Status, mR.Running())

		return mR, nil
	}

	return MigrationResponse{}, fmt.Errorf("migrations command failed after %d retries: %w", maxRetries, lastErr)
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	retryableErrors := []string{
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

	for _, retryable := range retryableErrors {
		if strings.Contains(errStr, retryable) {
			return true
		}
	}

	if strings.Contains(errStr, "server error") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "504") {
		return true
	}

	return false
}

type MigrationResponse struct {
	Success   bool            `json:"success"`
	Status    string          `json:"status"`
	Output    string          `json:"output"`
	Exception string          `json:"exception"`
	Stats     json.RawMessage `json:"stats"`
}

func (mR MigrationResponse) GetOutput(command string) (string, error) {
	if mR.Faulty() {
		return mR.formatAll()
	}
	if command == "stats" {
		return mR.formatStats()
	}
	return mR.Output, nil
}

func (mR MigrationResponse) formatStats() (string, error) {
	var stats map[string]any
	if err := json.Unmarshal(mR.Stats, &stats); err != nil {
		return "", fmt.Errorf("unmarshalling stats: %w", err)
	}

	// Define the order we want fields printed
	orderedFields := []string{
		"current_migration_index",
		"target_migration_index",
		"positions",
		"events",
		"partially_migrated_positions",
		"fully_migrated_positions",
		"status",
	}

	var sb strings.Builder
	for _, field := range orderedFields {
		if value, ok := stats[field]; ok {
			sb.WriteString(fmt.Sprintf("%s: %v\n", field, value))
		}
	}

	return sb.String(), nil
}

func (mR MigrationResponse) formatAll() (string, error) {
	return fmt.Sprintf("Success: %v\nStatus: %s\nOutput: %s\nException: %s\n",
		mR.Success, mR.Status, mR.Output, mR.Exception), nil
}

func (mR MigrationResponse) Faulty() bool {
	return !mR.Success || mR.Exception != ""
}

func (mR MigrationResponse) Running() bool {
	return mR.Status == migrationRunning
}
