package main

import (
	"fmt"
	"os"

	"github.com/OpenSlides/openslides-cli/internal/instance/config"
	"github.com/OpenSlides/openslides-cli/internal/instance/create"
	"github.com/OpenSlides/openslides-cli/internal/instance/remove"
	"github.com/OpenSlides/openslides-cli/internal/instance/setup"
	k8sActions "github.com/OpenSlides/openslides-cli/internal/k8s/actions"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/OpenSlides/openslides-cli/internal/manage/actions/action"
	"github.com/OpenSlides/openslides-cli/internal/manage/actions/createuser"
	"github.com/OpenSlides/openslides-cli/internal/manage/actions/get"
	"github.com/OpenSlides/openslides-cli/internal/manage/actions/initialdata"
	"github.com/OpenSlides/openslides-cli/internal/manage/actions/migrations"
	"github.com/OpenSlides/openslides-cli/internal/manage/actions/set"
	"github.com/OpenSlides/openslides-cli/internal/manage/actions/setpassword"

	"github.com/spf13/cobra"
)

const RootHelp = `osmanage is an admin tool to perform management actions on OpenSlides instances.`

func main() {
	code := RunClient()
	os.Exit(code)
}

func RunClient() int {
	err := RootCmd().Execute()

	if err == nil {
		return 0
	}

	code := 1
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)

	return code
}

func RootCmd() *cobra.Command {
	var logLevel string

	rootCmd := &cobra.Command{
		Use:               "osmanage",
		Short:             "Swiss army knife for OpenSlides admins",
		Long:              RootHelp,
		SilenceErrors:     true,
		SilenceUsage:      true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}

	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "warn", "Log level (debug, info, warn, error)")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		log, err := logger.New(logLevel)
		if err != nil {
			return fmt.Errorf("invalid log level: %w", err)
		}
		logger.SetGlobal(log)
		logger.Debug("Logger initialized at level: %s", logLevel)
		return nil
	}

	// K8s command group
	k8sCmd := &cobra.Command{
		Use:   "k8s",
		Short: "Manage Kubernetes deployments",
		Long:  "Manage OpenSlides instances deployed on Kubernetes",
	}

	k8sCmd.AddCommand(
		k8sActions.StartCmd(),
		k8sActions.StopCmd(),
		k8sActions.HealthCmd(),
		k8sActions.ClusterStatusCmd(),
		k8sActions.UpdateBackendmanageCmd(),
		k8sActions.UpdateInstanceCmd(),
		k8sActions.ScaleCmd(),
	)

	rootCmd.AddCommand(
		setup.Cmd(),
		config.Cmd(),
		create.Cmd(),
		remove.Cmd(),
		createuser.Cmd(),
		initialdata.Cmd(),
		setpassword.Cmd(),
		set.Cmd(),
		get.Cmd(),
		action.Cmd(),
		migrations.Cmd(),
		k8sCmd,
	)

	return rootCmd
}
