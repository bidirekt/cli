package publish_contract

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bidirekt/cli/internal/contractfiles"
	"github.com/bidirekt/cli/internal/paint"
	"github.com/bidirekt/cli/internal/reports"
	"github.com/spf13/cobra"
)

const requestTimeout = 30 * time.Second

func NewPublishCommand(publishContractClient *PublishContractClient) *cobra.Command {

	commandHandler := func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true
		command.SilenceErrors = true

		contracts, err := contractfiles.ToContractFragments(args)
		if err != nil {
			return err
		}

		participant, err := command.Flags().GetString("participant")
		if err != nil {
			return fmt.Errorf("get participant: %w", err)
		}

		version, err := command.Flags().GetString("version")
		if err != nil {
			return fmt.Errorf("get version: %w", err)
		}

		ctx, cancel := context.WithTimeout(command.Context(), requestTimeout)
		defer cancel()

		requestBody := &PublishContractRequestBody{
			Participant: participant,
			Version:     version,
			Contracts:   contracts,
		}

		message, err := publishContractClient.PublishContract(ctx, requestBody)
		if err != nil {
			var validationFailed *reports.ValidationFailedError
			if errors.As(err, &validationFailed) {
				errWriter := command.ErrOrStderr()
				if _, err := fmt.Fprint(errWriter, reports.FormatValidationFailedReport(paint.For(errWriter), validationFailed.Message, validationFailed.Violations)); err != nil {
					return err
				}

				return reports.ErrSilent
			}

			return err
		}

		writer := command.OutOrStdout()
		brush := paint.For(writer)
		if _, err := fmt.Fprintln(writer, brush.Green(participant+" "+message)); err != nil {
			return err
		}

		return nil
	}

	command := &cobra.Command{
		Use:   "publish [file...]",
		Short: "Publish one or more contract YAML files to the broker",
		Args:  cobra.MinimumNArgs(1),
		RunE:  commandHandler,
	}

	command.Flags().String("participant", "", "Participant name (required)")
	command.Flags().String("version", "", "Contract version, e.g. a commit hash or semver tag (required)")
	_ = command.MarkFlagRequired("participant")
	_ = command.MarkFlagRequired("version")

	return command
}
