package logout

import (
	"fmt"
	"strings"

	"github.com/bidirekt/cli/internal/components"
	"github.com/spf13/cobra"
)

func NewLogoutCommand(config *components.Config) *cobra.Command {
	commandHandler := func(command *cobra.Command, _ []string) error {
		flagProfile, err := command.Flags().GetString("profile")
		if err != nil {
			return fmt.Errorf("get profile: %w", err)
		}

		configFile, err := components.ConfigFileFromEnvironment()
		if err != nil {
			return err
		}
		profileName := components.ActiveProfileName(flagProfile, config.Profile)

		profiles, err := configFile.ReadProfiles()
		if err != nil {
			return err
		}
		brokerURL := profiles[profileName].BrokerURL

		if profiles[profileName].APIKey == "" {
			_, err = fmt.Fprintf(command.OutOrStdout(), "not signed in on profile %s\n", profileName)
		} else {
			err = signOut(command, configFile, profileName, brokerURL)
		}
		if err != nil {
			return err
		}

		if config.BrokerURL != "" && config.APIKey != "" {
			_, err = fmt.Fprintln(command.ErrOrStderr(), "BIDIREKT_BROKER_URL and BIDIREKT_API_KEY are still set and keep signing commands in")
		}
		return err
	}

	command := &cobra.Command{
		Use:   "logout",
		Short: "Remove the API key from a profile in the config file",
		Args:  cobra.NoArgs,
		RunE:  commandHandler,
	}

	command.Flags().String("profile", "", `Profile to sign out of (falls back to BIDIREKT_PROFILE, then "default")`)

	return command
}

func signOut(command *cobra.Command, configFile components.ConfigFile, profileName, brokerURL string) error {
	if err := configFile.WriteProfile(profileName, func(stored *components.Profile) { stored.APIKey = "" }); err != nil {
		return err
	}

	_, err := fmt.Fprintf(command.OutOrStdout(), "Signed out of %s (profile %s). The key is still valid — revoke it at %s/profile\n",
		components.RedactedURL(brokerURL), profileName, components.RedactedURL(strings.TrimSuffix(brokerURL, "/")))
	return err
}
