package list_environments

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

const requestTimeout = 30 * time.Second

func NewListEnvironmentsCommand(client *ListEnvironmentsClient) *cobra.Command {
	commandHandler := func(command *cobra.Command, _ []string) error {
		command.SilenceUsage = true

		ctx, cancel := context.WithTimeout(command.Context(), requestTimeout)
		defer cancel()

		names, err := client.List(ctx)
		if err != nil {
			return err
		}

		writer := command.OutOrStdout()
		for _, name := range names {
			if _, err := fmt.Fprintln(writer, name); err != nil {
				return err
			}
		}

		return nil
	}

	command := &cobra.Command{
		Use:   "list-environments",
		Short: "List the environments on the broker",
		Args:  cobra.NoArgs,
		RunE:  commandHandler,
	}

	return command
}
