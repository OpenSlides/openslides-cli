package setpassword

import (
	"encoding/json"
	"fmt"

	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/OpenSlides/openslides-cli/internal/manage/client"
	"github.com/OpenSlides/openslides-cli/internal/utils"

	"github.com/spf13/cobra"
)

const (
	SetPasswordHelp      = "Sets the password of a user in OpenSlides"
	SetPasswordHelpExtra = "This command sets the password of a user by a given user ID."
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-password",
		Short: SetPasswordHelp,
		Long:  SetPasswordHelp + "\n\n" + SetPasswordHelpExtra,
		Args:  cobra.NoArgs,
	}

	address := cmd.Flags().StringP("address", "a", "localhost:9002", "address of the OpenSlides backendManage service")
	passwordFile := cmd.Flags().String("password-file", "secrets/internal_auth_password", "file with password for authorization")
	userID := cmd.Flags().Int64P("user_id", "u", 0, "ID of the user account")
	password := cmd.Flags().StringP("password", "p", "", "new password of the user")

	_ = cmd.MarkFlagRequired("user_id")
	_ = cmd.MarkFlagRequired("password")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if *password == "" {
			return fmt.Errorf("--password cannot be empty")
		}

		logger.Info("=== SET PASSWORD ===")
		logger.Debug("Setting password for user ID: %d", *userID)

		authPassword, err := utils.ReadPassword(*passwordFile)
		if err != nil {
			return fmt.Errorf("reading password: %w", err)
		}

		payload := []map[string]any{
			{
				"id":       *userID,
				"password": *password,
			},
		}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshalling payload: %w", err)
		}

		cl := client.New(*address, authPassword)
		resp, err := cl.SendAction("user.set_password", payloadJSON)
		if err != nil {
			return fmt.Errorf("sending request: %w", err)
		}

		body, err := client.CheckResponse(resp)
		if err != nil {
			return err
		}

		logger.Info("Password set successfully for user %d", *userID)
		fmt.Printf("Response: %s\n", string(body))
		fmt.Printf("Password for user %d set successfully.\n", *userID)
		return nil
	}

	return cmd
}
