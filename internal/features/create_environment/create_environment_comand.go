package create_environment

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

const requestTimeout = 30 * time.Second

func NewCreateEnvironmentCommand(client *CreateEnvironmentClient) *cobra.Command {
	commandHandler := func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		name := args[0]
		if name == "" {
			return fmt.Errorf("environment name must not be empty")
		}

		ctx, cancel := context.WithTimeout(command.Context(), requestTimeout)
		defer cancel()

		requestBody := &CreateEnvironmentRequestBody{
			Environment: name,
		}

		message, err := client.Create(ctx, requestBody)
		if err != nil {
			return err
		}

		if _, err := fmt.Fprintf(command.OutOrStdout(), "🌍 %s %s\n", name, message); err != nil {
			return err
		}

		return nil
	}

	command := &cobra.Command{
		Use:   "create-environment [name]",
		Short: "Create a new environment on the broker",
		Args:  cobra.ExactArgs(1),
		RunE:  commandHandler,
	}

	return command
}
