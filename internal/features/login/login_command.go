package login

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/bidirekt/cli/internal/components"
	"github.com/spf13/cobra"
)

const (
	requestTimeout = 30 * time.Second
	apiKeyPrefix   = "bdk_"
)

var (
	errNoTerminalToSignIn = errors.New("no terminal to sign in — set BIDIREKT_BROKER_URL and BIDIREKT_API_KEY")
	errNotAnAPIKey        = errors.New("an API key starts with " + apiKeyPrefix)
)

func NewLoginCommand(config *components.Config, client *LoginClient, isTerminal func(stream any) bool, readSecret func(stream any) (string, error)) *cobra.Command {
	commandHandler := func(command *cobra.Command, _ []string) error {
		command.SilenceUsage = true

		stdin := command.InOrStdin()
		stderr := command.ErrOrStderr()
		if !isTerminal(stdin) {
			return errNoTerminalToSignIn
		}

		flagProfile, err := command.Flags().GetString("profile")
		if err != nil {
			return fmt.Errorf("get profile: %w", err)
		}

		lines := bufio.NewScanner(stdin)
		profile := flagProfile
		if profile == "" {
			profile, err = promptProfile(lines, stderr, components.ActiveProfileName("", config.Profile))
			if err != nil {
				return err
			}
		}

		configFile, err := components.ConfigFileFromEnvironment()
		if err != nil {
			return err
		}
		profiles, err := configFile.ReadProfiles()
		if err != nil {
			return err
		}
		previous := profiles[profile]

		brokerURL, err := components.PromptBrokerURL(lines, stderr, previous.BrokerURL)
		if err != nil {
			return err
		}

		if _, err := fmt.Fprintf(stderr, "Create an API key at %s and paste it here.\n", link(brokerURL, "/profile/api-keys/new")); err != nil {
			return err
		}
		if _, err := fmt.Fprint(stderr, "API key: "); err != nil {
			return err
		}
		apiKey, err := readSecret(stdin)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(stderr); err != nil {
			return err
		}
		if !strings.HasPrefix(apiKey, apiKeyPrefix) {
			return errNotAnAPIKey
		}

		if err := components.WarnIfKeyTravelsUnencrypted(stderr, brokerURL, apiKey); err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(command.Context(), requestTimeout)
		defer cancel()
		if err := client.CheckKey(ctx, brokerURL, apiKey); err != nil {
			return err
		}

		if err := configFile.WriteProfile(profile, func(stored *components.Profile) {
			stored.BrokerURL = brokerURL
			stored.APIKey = apiKey
		}); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(command.OutOrStdout(), "Signed in to %s as profile %s\n", components.RedactedURL(brokerURL), profile); err != nil {
			return err
		}

		return warnAfterSignIn(stderr, config, previous, apiKey)
	}

	command := &cobra.Command{
		Use:   "login",
		Short: "Sign in to a broker with an API key and save both in a profile",
		Args:  cobra.NoArgs,
		RunE:  commandHandler,
	}

	command.Flags().String("profile", "", `Profile to sign in (asked when not given)`)

	return command
}

func warnAfterSignIn(stderr io.Writer, config *components.Config, previous components.Profile, apiKey string) error {
	var warnings []string
	if previous.APIKey != "" && previous.APIKey != apiKey {
		warnings = append(warnings, "the previous key is still valid — revoke it at "+link(previous.BrokerURL, "/profile"))
	}
	switch {
	case config.BrokerURL != "" && config.APIKey != "":
		warnings = append(warnings, "BIDIREKT_BROKER_URL and BIDIREKT_API_KEY are set and win over this profile")
	case config.BrokerURL != "" || config.APIKey != "":
		warnings = append(warnings, "only one of BIDIREKT_BROKER_URL and BIDIREKT_API_KEY is set — commands will fail until both or neither are")
	}

	for _, warning := range warnings {
		if _, err := fmt.Fprintln(stderr, warning); err != nil {
			return err
		}
	}
	return nil
}

func link(brokerURL, path string) string {
	return components.RedactedURL(strings.TrimSuffix(brokerURL, "/")) + path
}
