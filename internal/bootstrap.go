package internal

import (
	"errors"
	"fmt"
	"os"

	"github.com/contracttesting/cli/internal/components"
	"github.com/contracttesting/cli/internal/features/can_i_deploy"
	"github.com/contracttesting/cli/internal/features/create_environment"
	"github.com/contracttesting/cli/internal/features/create_participant"
	"github.com/contracttesting/cli/internal/features/publish_contract"
	"github.com/contracttesting/cli/internal/features/record_deployment"
	"github.com/contracttesting/cli/internal/features/rename_participant"
	"github.com/spf13/cobra"
)

// Overridden at link time by the release pipeline (-ldflags "-X github.com/contracttesting/cli/internal.version=<tag>").
var version = "dev"

var rootCommand = &cobra.Command{
	Use:           "ctio",
	Short:         "CLI for ContractTesting",
	Version:       version,
	SilenceErrors: true,
	SilenceUsage:  true,
}

var versionCommand = &cobra.Command{
	Use:   "version",
	Short: "Print the ctio version",
	Run: func(command *cobra.Command, _ []string) {
		_, _ = fmt.Fprintf(command.OutOrStdout(), "ctio version %s\n", version)
	},
}

func Run() {
	components := components.New()

	rootCommand.
		PersistentFlags().
		String(
			"broker-url",
			components.Config.BrokerURL,
			"Broker base URL",
		)

	rootCommand.PersistentPreRunE = func(command *cobra.Command, _ []string) error {
		brokerURL, err := command.Flags().GetString("broker-url")
		if err != nil {
			return err
		}
		components.HTTPClient.SetBaseURL(brokerURL)

		return nil
	}

	create_participant.Register(rootCommand, components)
	create_environment.Register(rootCommand, components)
	publish_contract.Register(rootCommand, components)
	record_deployment.Register(rootCommand, components)
	can_i_deploy.Register(rootCommand, components)
	rename_participant.Register(rootCommand, components)
	rootCommand.AddCommand(versionCommand)

	if err := rootCommand.Execute(); err != nil {
		if !errors.Is(err, can_i_deploy.ErrSilent) && !errors.Is(err, publish_contract.ErrSilent) {
			_, _ = fmt.Fprintf(rootCommand.ErrOrStderr(), "❌ %s\n", err.Error())
		}

		os.Exit(1)
	}
}
