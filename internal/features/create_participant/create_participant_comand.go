package create_participant

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

const requestTimeout = 30 * time.Second

func NewCreateParticipantCommand(client *CreateParticipantClient) *cobra.Command {
	commandHandler := func(command *cobra.Command, args []string) error {
		name := args[0]
		if name == "" {
			return fmt.Errorf("participant name must not be empty")
		}

		ctx, cancel := context.WithTimeout(command.Context(), requestTimeout)
		defer cancel()

		requestBody := &CreateParticipantRequestBody{
			Participant: name,
		}

		message, err := client.Create(ctx, requestBody)
		if err != nil {
			return err
		}

		if _, err := fmt.Fprintf(command.OutOrStdout(), "%s %s\n", name, message); err != nil {
			return err
		}

		return nil
	}

	command := &cobra.Command{
		Use:   "create-participant [name]",
		Short: "Create a new participant on the broker",
		Args:  cobra.ExactArgs(1),
		RunE:  commandHandler,
	}

	return command
}
