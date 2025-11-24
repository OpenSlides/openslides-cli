package initialdata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/OpenSlides/openslides-cli/internal/client"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/OpenSlides/openslides-cli/internal/utils"

	"github.com/spf13/cobra"
)

const (
	InitialDataHelp      = "Creates initial data if the datastore is empty"
	InitialDataHelpExtra = `This command sets up initial data for a new OpenSlides instance.
Provide initial data via --file flag with a JSON file path, or use --file=- to read from stdin.
If no file is provided, empty initialization data will be used.

This command also sets the superadmin (user 1) password from the superadmin password file.
It returns an error if the datastore is not empty.`
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "initial-data",
		Short: InitialDataHelp,
		Long:  InitialDataHelp + "\n\n" + InitialDataHelpExtra,
		Args:  cobra.NoArgs,
	}

	address := cmd.Flags().StringP("address", "a", "localhost:9002", "address of the OpenSlides backendManage service")
	passwordFile := cmd.Flags().String("password-file", "secrets/internal_auth_password", "file with password for authorization")
	superadminPasswordFile := cmd.Flags().String("superadmin-password-file", "secrets/superadmin", "file with superadmin password")
	dataFile := cmd.Flags().StringP("file", "f", "", "JSON file with initial data, or - for stdin")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		logger.Info("=== INITIAL DATA ===")

		var data []byte
		var err error

		if *dataFile != "" {
			data, err = utils.ReadFromFileOrStdin(*dataFile)
			if err != nil {
				return fmt.Errorf("reading initial data: %w", err)
			}
		}

		if len(data) == 0 {
			logger.Debug("No data provided, using empty object")
			data = []byte("{}")
		}

		password, err := utils.ReadPassword(*passwordFile)
		if err != nil {
			return fmt.Errorf("reading password: %w", err)
		}

		payload := []map[string]any{
			{
				"data": json.RawMessage(data),
			},
		}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshalling payload: %w", err)
		}

		cl := client.New(*address, password)
		resp, err := cl.SendAction("organization.initial_import", payloadJSON)
		if err != nil {
			return fmt.Errorf("sending request: %w", err)
		}

		body, err := client.CheckResponse(resp)
		if err != nil {
			if bytes.Contains(body, []byte("Datastore is not empty")) {
				logger.Warn("Datastore is not empty")
				fmt.Fprintln(os.Stderr, "Datastore contains data, initial data were NOT set")
				os.Exit(2)
			}
			return err
		}

		logger.Info("Initial data set successfully")
		fmt.Println("Initial data set successfully.")

		if err := setSuperadminPassword(*address, password, *superadminPasswordFile); err != nil {
			return fmt.Errorf("setting superadmin password: %w", err)
		}

		logger.Info("Superadmin password set successfully")
		fmt.Println("Superadmin password set successfully.")
		return nil
	}

	return cmd
}

func setSuperadminPassword(address, authPassword, superadminPasswordFile string) error {
	logger.Debug("Setting superadmin password")

	superadminPW, err := utils.ReadPassword(superadminPasswordFile)
	if err != nil {
		return fmt.Errorf("reading superadmin password: %w", err)
	}

	payload := []map[string]any{
		{
			"id":       1,
			"password": superadminPW,
		},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshalling password payload: %w", err)
	}

	cl := client.New(address, authPassword)
	resp, err := cl.SendAction("user.set_password", payloadJSON)
	if err != nil {
		return fmt.Errorf("sending password request: %w", err)
	}

	_, err = client.CheckResponse(resp)
	return err
}
