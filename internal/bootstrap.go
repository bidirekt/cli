package internal

import (
	"errors"
	"fmt"
	"os"

	"github.com/bidirekt/cli/internal/components"
	"github.com/bidirekt/cli/internal/features/can_i_deploy"
	"github.com/bidirekt/cli/internal/features/configure"
	"github.com/bidirekt/cli/internal/features/create_environment"
	"github.com/bidirekt/cli/internal/features/create_participant"
	"github.com/bidirekt/cli/internal/features/publish_contract"
	"github.com/bidirekt/cli/internal/features/record_deployment"
	"github.com/bidirekt/cli/internal/features/rename_participant"
	"github.com/bidirekt/cli/internal/paint"
	"github.com/spf13/cobra"
)

// Overridden at link time by the release pipeline (-ldflags "-X github.com/bidirekt/cli/internal.version=<tag>").
var version = "dev"

var errNoBrokerConfigured = errors.New(`no broker configured — pass --broker-url, set BIDIREKT_BROKER_URL, or run "bidirekt configure"`)

func Run() {
	rootCommand := newRootCommand(components.New(), components.IsTerminal)

	if err := rootCommand.Execute(); err != nil {
		if !errors.Is(err, can_i_deploy.ErrSilent) && !errors.Is(err, publish_contract.ErrSilent) {
			errWriter := rootCommand.ErrOrStderr()
			_, _ = fmt.Fprintln(errWriter, paint.For(errWriter).Red(err.Error()))
		}

		os.Exit(1)
	}
}

func newRootCommand(dependencies *components.Components, isTerminal func(stream any) bool) *cobra.Command {
	rootCommand := &cobra.Command{
		Use:           "bidirekt",
		Short:         "CLI for Bidirekt",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(command *cobra.Command, _ []string) error {
			if _, talksToBroker := command.Annotations[components.TalksToBrokerAnnotation]; !talksToBroker {
				return nil
			}

			if err := command.ValidateRequiredFlags(); err != nil {
				return err
			}
			if err := command.ValidateFlagGroups(); err != nil {
				return err
			}

			broker, err := resolveBroker(command, dependencies.Config, isTerminal)
			if err != nil {
				return err
			}

			if _, err := fmt.Fprintf(command.ErrOrStderr(), "Broker: %s (%s)\n", components.RedactedURL(broker.BrokerURL), broker.Origin); err != nil {
				return err
			}
			dependencies.HTTPClient.SetBaseURL(broker.BrokerURL)

			return nil
		},
	}

	rootCommand.PersistentFlags().String("broker-url", "", "Broker base URL")
	rootCommand.PersistentFlags().String("profile", "", `Profile in the config file (falls back to BIDIREKT_PROFILE, then "default")`)

	create_participant.Register(rootCommand, dependencies)
	create_environment.Register(rootCommand, dependencies)
	publish_contract.Register(rootCommand, dependencies)
	record_deployment.Register(rootCommand, dependencies)
	can_i_deploy.Register(rootCommand, dependencies)
	rename_participant.Register(rootCommand, dependencies)
	configure.Register(rootCommand, dependencies)
	rootCommand.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the bidirekt version",
		Run: func(command *cobra.Command, _ []string) {
			_, _ = fmt.Fprintf(command.OutOrStdout(), "bidirekt version %s\n", version)
		},
	})

	return rootCommand
}

func resolveBroker(command *cobra.Command, config *components.Config, isTerminal func(stream any) bool) (components.ResolvedBrokerURL, error) {
	flagBrokerURL, err := command.Flags().GetString("broker-url")
	if err != nil {
		return components.ResolvedBrokerURL{}, err
	}

	flagProfile, err := command.Flags().GetString("profile")
	if err != nil {
		return components.ResolvedBrokerURL{}, err
	}

	configFile, err := components.ConfigFileFromEnvironment()
	if err != nil {
		return components.ResolvedBrokerURL{}, err
	}

	resolved, err := components.ResolveBrokerURL(components.BrokerURLSources{
		FlagBrokerURL: flagBrokerURL,
		EnvBrokerURL:  config.BrokerURL,
		FlagProfile:   flagProfile,
		EnvProfile:    config.Profile,
		ReadProfiles:  configFile.ReadProfiles,
	})
	if err != nil || resolved.BrokerURL != "" {
		return resolved, err
	}

	stdin := command.InOrStdin()
	if !isTerminal(stdin) {
		if resolved.ProfileNotFound {
			return components.ResolvedBrokerURL{}, fmt.Errorf("profile %q not found in %s", resolved.Profile, configFile.Path)
		}
		return components.ResolvedBrokerURL{}, errNoBrokerConfigured
	}

	resolved.BrokerURL, err = components.PromptBrokerURL(stdin, command.ErrOrStderr(), "")
	if err != nil {
		return components.ResolvedBrokerURL{}, err
	}

	if err := configFile.WriteProfile(resolved.Profile, components.Profile{BrokerURL: resolved.BrokerURL}); err != nil {
		return components.ResolvedBrokerURL{}, err
	}

	return resolved, nil
}
