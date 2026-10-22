package rename_participant

import (
	"context"
	"fmt"
	"time"

	"github.com/bidirekt/cli/internal/paint"
	"github.com/spf13/cobra"
)

const requestTimeout = 30 * time.Second

func NewRenameParticipantCommand(client *RenameParticipantClient) *cobra.Command {
	commandHandler := func(command *cobra.Command, args []string) error {
		oldName := args[0]
		newName := args[1]

		if oldName == "" || newName == "" {
			return fmt.Errorf("participant names must not be empty")
		}

		ctx, cancel := context.WithTimeout(command.Context(), requestTimeout)
		defer cancel()

		requestBody := &RenameParticipantRequestBody{
			OldName: oldName,
			NewName: newName,
		}

		message, err := client.Rename(ctx, requestBody)
		if err != nil {
			return err
		}

		writer := command.OutOrStdout()
		brush := paint.For(writer)
		if _, err := fmt.Fprintln(writer, brush.Green(oldName+" "+message+" to "+newName)); err != nil {
			return err
		}
		return nil
	}

	command := &cobra.Command{
		Use:   "rename-participant [old] [new]",
		Short: "Rename an existing participant on the broker",
		Args:  cobra.ExactArgs(2),
		RunE:  commandHandler,
	}

	return command
}
