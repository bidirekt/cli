package configure

import (
	"errors"
	"fmt"

	"github.com/bidirekt/cli/internal/components"
	"github.com/spf13/cobra"
)

var errNoTerminalToAsk = errors.New("no terminal to ask for the broker URL — pass --broker-url")

func NewConfigureCommand(config *components.Config, isTerminal func(stream any) bool) *cobra.Command {
	commandHandler := func(command *cobra.Command, _ []string) error {
		brokerURL, err := command.Flags().GetString("broker-url")
		if err != nil {
			return fmt.Errorf("get broker-url: %w", err)
		}

		flagProfile, err := command.Flags().GetString("profile")
		if err != nil {
			return fmt.Errorf("get profile: %w", err)
		}

		configFile, err := components.ConfigFileFromEnvironment()
		if err != nil {
			return err
		}
		profile := components.ActiveProfileName(flagProfile, config.Profile)

		switch {
		case brokerURL != "":
			err = components.ValidateBrokerURL(brokerURL, components.BrokerURLFlagOrigin)
		case isTerminal(command.InOrStdin()):
			brokerURL, err = promptBrokerURLOfProfile(command, configFile, profile)
		default:
			err = errNoTerminalToAsk
		}
		if err != nil {
			return err
		}

		return configFile.WriteProfile(profile, components.Profile{BrokerURL: brokerURL})
	}

	command := &cobra.Command{
		Use:   "configure",
		Short: "Save the broker URL of a profile in the config file",
		Args:  cobra.NoArgs,
		RunE:  commandHandler,
	}

	command.Flags().String("broker-url", "", "Broker URL to save without asking")
	command.Flags().String("profile", "", `Profile to save into (falls back to BIDIREKT_PROFILE, then "default")`)

	return command
}

func promptBrokerURLOfProfile(command *cobra.Command, configFile components.ConfigFile, profile string) (string, error) {
	profiles, err := configFile.ReadProfiles()
	if err != nil {
		return "", err
	}

	return components.PromptBrokerURL(command.InOrStdin(), command.ErrOrStderr(), profiles[profile].BrokerURL)
}
