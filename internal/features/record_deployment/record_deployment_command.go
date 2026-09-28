package record_deployment

import (
	"context"
	"fmt"
	"time"

	"github.com/bidirekt/cli/internal/paint"
	"github.com/spf13/cobra"
)

const requestTimeout = 30 * time.Second

func NewRecordDeploymentCommand(client *RecordDeploymentClient) *cobra.Command {
	commandHandler := func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		participant := args[0]

		version, err := command.Flags().GetString("version")
		if err != nil {
			return fmt.Errorf("get version: %w", err)
		}

		environment, err := command.Flags().GetString("environment")
		if err != nil {
			return fmt.Errorf("get environment: %w", err)
		}

		ctx, cancel := context.WithTimeout(command.Context(), requestTimeout)
		defer cancel()

		requestBody := &RecordDeploymentRequestBody{
			Participant: participant,
			Version:     version,
			Environment: environment,
		}

		message, err := client.Record(ctx, requestBody)
		if err != nil {
			return err
		}

		writer := command.OutOrStdout()
		brush := paint.For(writer)
		if _, err := fmt.Fprintln(writer, brush.Green(participant+" "+message+" to "+environment)); err != nil {
			return err
		}
		return nil
	}

	command := &cobra.Command{
		Use:   "record-deployment [participant]",
		Short: "Record a deployment of a participant version to an environment",
		Args:  cobra.ExactArgs(1),
		RunE:  commandHandler,
	}

	command.Flags().String("version", "", "Deployed version, e.g. a commit hash or semver tag (required)")
	command.Flags().String("environment", "", "Target environment name (required)")
	_ = command.MarkFlagRequired("version")
	_ = command.MarkFlagRequired("environment")

	return command
}
