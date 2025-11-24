package createuser

import (
	"encoding/json"
	"fmt"

	"github.com/OpenSlides/openslides-cli/internal/client"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/OpenSlides/openslides-cli/internal/utils"

	"github.com/spf13/cobra"
)

const (
	CreateUserHelp      = "Creates a new user in OpenSlides"
	CreateUserHelpExtra = `This command creates a new user with the given user data in JSON format.
Provide the user data as an argument, or use the --file flag with a file path,
or use --file=- to read from stdin.`
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-user [user-data]",
		Short: CreateUserHelp,
		Long:  CreateUserHelp + "\n\n" + CreateUserHelpExtra,
		Args:  cobra.RangeArgs(0, 1),
	}

	address := cmd.Flags().StringP("address", "a", "localhost:9002", "address of the OpenSlides backendManage service")
	passwordFile := cmd.Flags().String("password-file", "secrets/internal_auth_password", "file with password for authorization")
	userFile := cmd.Flags().StringP("file", "f", "", "JSON file with user data, or - for stdin")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		logger.Info("=== CREATE USER ===")

		var input string
		if len(args) > 0 {
			input = args[0]
		}

		userData, err := utils.ReadInputOrFileOrStdin(input, *userFile)
		if err != nil {
			return fmt.Errorf("reading user data: %w", err)
		}

		var userPayload map[string]any
		if err := json.Unmarshal(userData, &userPayload); err != nil {
			logger.Error("Invalid JSON in user data")
			return fmt.Errorf("invalid JSON: %w", err)
		}

		logger.Debug("Parsed user data: %v", userPayload)

		if userPayload["username"] == nil || userPayload["default_password"] == nil {
			return fmt.Errorf("missing required fields: username and default_password")
		}

		password, err := utils.ReadPassword(*passwordFile)
		if err != nil {
			return fmt.Errorf("reading password: %w", err)
		}

		userDataArray := []map[string]any{userPayload}
		userDataJSON, err := json.Marshal(userDataArray)
		if err != nil {
			return fmt.Errorf("marshalling user data: %w", err)
		}

		cl := client.New(*address, password)
		resp, err := cl.SendAction("user.create", userDataJSON)
		if err != nil {
			return fmt.Errorf("sending request: %w", err)
		}

		body, err := client.CheckResponse(resp)
		if err != nil {
			return err
		}

		logger.Info("User created successfully")
		fmt.Printf("Response: %s\n", string(body))
		return nil
	}

	return cmd
}
