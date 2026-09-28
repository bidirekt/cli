package create_participant

import (
	"context"
	"fmt"
	"time"

	"github.com/bidirekt/cli/internal/paint"
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

		writer := command.OutOrStdout()
		brush := paint.For(writer)
		if _, err := fmt.Fprintln(writer, brush.Green(name+" "+message)); err != nil {
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
