package set

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/OpenSlides/openslides-cli/internal/manage/client"
	"github.com/OpenSlides/openslides-cli/internal/utils"

	"github.com/spf13/cobra"
)

const (
	SetHelp      = "Calls an OpenSlides action to update some objects"
	SetHelpExtra = `This command calls an OpenSlides backend action with the given JSON
formatted payload. Provide the payload directly or use the --file flag with a
file or use this flag with - to read from stdin. Only the following update actions are
supported: [agenda_item, committee, group, meeting, motion, organization_tag, organization, projector, theme, topic, user]

Examples:
  osmanage set user '[{"id": 5, "first_name": "Jane", "last_name": "Smith"}]'
	--address <myBackendManageIP>:9002 \
	--password-file ./my.instance.dir.org/secrets/internal_auth_password

  osmanage set user \
    --file user.json \
	--address <myBackendManageIP>:9002 \
	--password-file ./my.instance.dir.org/secrets/internal_auth_password

  echo '[{"id": 5, "first_name": "Jane", "last_name": "Smith"}]' | osmanage set user \
    --file - \
	--address <myBackendManageIP>:9002 \
	--password-file ./my.instance.dir.org/secrets/internal_auth_password`
)

var actionMap = map[string]string{
	"agenda_item":      "agenda_item.update",
	"committee":        "committee.update",
	"group":            "group.update",
	"meeting":          "meeting.update",
	"motion":           "motion.update",
	"organization_tag": "organization_tag.update",
	"organization":     "organization.update",
	"projector":        "projector.update",
	"theme":            "theme.update",
	"topic":            "topic.update",
	"user":             "user.update",
}

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set action [payload]",
		Short: SetHelp,
		Long:  SetHelp + "\n\n" + SetHelpExtra + strings.Join(helpTextActionList(), "\n    "),
		Args:  cobra.RangeArgs(1, 2),
	}

	address := cmd.Flags().StringP("address", "a", "", "address of the OpenSlides backendManage service (required)")
	passwordFile := cmd.Flags().String("password-file", "", "file with password for authorization (required)")
	payloadFile := cmd.Flags().StringP("file", "f", "", "JSON file with the payload, or - for stdin")

	_ = cmd.MarkFlagRequired("address")
	_ = cmd.MarkFlagRequired("password-file")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		logger.Info("=== SET ACTION ===")

		action := args[0]
		logger.Debug("Action type: %s", action)

		actionName, ok := actionMap[action]
		if !ok {
			logger.Error("Unknown action: %s", action)
			return fmt.Errorf("unknown action %q (available: %s)", action, strings.Join(helpTextActionList(), ", "))
		}

		logger.Debug("Mapped to backend action: %s", actionName)

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

func helpTextActionList() []string {
	actions := make([]string, 0, len(actionMap))
	for a := range actionMap {
		actions = append(actions, a)
	}
	sort.Strings(actions)
	return actions
}
