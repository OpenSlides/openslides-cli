package createuser

import (
	"encoding/json"
	"fmt"

	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/OpenSlides/openslides-cli/internal/manage/client"
	"github.com/OpenSlides/openslides-cli/internal/utils"

	"github.com/spf13/cobra"
)

const (
	CreateUserHelp      = "Creates a new user in OpenSlides"
	CreateUserHelpExtra = `This command creates a new user with the given user data in JSON format.
Provide the user data as an argument, or use the --file flag with a file path,
or use --file=- to read from stdin.

Examples:
  osmanage create-user '{"username": "myuser", "default_password": "mypwd"}' \
    --address <myBackendManageIP>:9002 \
    --password-file ./my.instance.dir.org/secrets/internal_auth_password

  osmanage create-user
    --file user.json \
    --address <myBackendManageIP>:9002 \
    --password-file ./my.instance.dir.org/secrets/internal_auth_password

  echo '{"username": "myuser", "default_password": "mypwd"}' | osmanage create-user \
    --file - \
    --address <myBackendManageIP>:9002 \
    --password-file ./my.instance.dir.org/secrets/internal_auth_password
`
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-user [user-data]",
		Short: CreateUserHelp,
		Long:  CreateUserHelp + "\n\n" + CreateUserHelpExtra,
		Args:  cobra.RangeArgs(0, 1),
	}

	address := cmd.Flags().StringP("address", "a", "", "address of the OpenSlides backendManage service (required)")
	passwordFile := cmd.Flags().String("password-file", "", "file with password for authorization (required)")
	userFile := cmd.Flags().StringP("file", "f", "", "JSON file with user data, or - for stdin")

	_ = cmd.MarkFlagRequired("address")
	_ = cmd.MarkFlagRequired("password-file")

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
