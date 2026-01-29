package action

import (
	"encoding/json"
	"fmt"

	"github.com/OpenSlides/openslides-cli/internal/manage/client"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/OpenSlides/openslides-cli/internal/utils"

	"github.com/spf13/cobra"
)

const (
	ActionHelp      = "Calls an arbitrary OpenSlides action"
	ActionHelpExtra = `This command calls an OpenSlides backend action with the given JSON
formatted payload. Provide the payload directly or use the --file flag with a
file or use this flag with - to read from stdin.`
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "action name [payload]",
		Short: ActionHelp,
		Long:  ActionHelp + "\n\n" + ActionHelpExtra,
		Args:  cobra.RangeArgs(1, 2),
	}

	address := cmd.Flags().StringP("address", "a", "localhost:9002", "address of the OpenSlides backendManage service")
	passwordFile := cmd.Flags().String("password-file", "secrets/internal_auth_password", "file with password for authorization")
	payloadFile := cmd.Flags().StringP("file", "f", "", "JSON file with the payload, or - for stdin")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		logger.Info("=== ACTION ===")

		actionName := args[0]
		logger.Debug("Action name: %s", actionName)

		var input string
		if len(args) > 1 {
			input = args[1]
		}

		payload, err := utils.ReadInputOrFileOrStdin(input, *payloadFile)
		if err != nil {
			return fmt.Errorf("reading payload: %w", err)
		}

		var payloadData any
		if err := json.Unmarshal(payload, &payloadData); err != nil {
			logger.Error("Invalid JSON in payload")
			return fmt.Errorf("invalid JSON: %w", err)
		}

		authPassword, err := utils.ReadPassword(*passwordFile)
		if err != nil {
			return fmt.Errorf("reading password: %w", err)
		}

		cl := client.New(*address, authPassword)
		resp, err := cl.SendAction(actionName, payload)
		if err != nil {
			return fmt.Errorf("sending request: %w", err)
		}

		body, err := client.CheckResponse(resp)
		if err != nil {
			return err
		}

		logger.Info("Action completed successfully")
		fmt.Printf("Request was successful with following response: %s\n", string(body))
		return nil
	}

	return cmd
}
