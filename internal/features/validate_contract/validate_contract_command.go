package validate_contract

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

func NewValidateCommand(validateContractClient *ValidateContractClient) *cobra.Command {
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

		environment, err := command.Flags().GetString("environment")
		if err != nil {
			return fmt.Errorf("get environment: %w", err)
		}

		ctx, cancel := context.WithTimeout(command.Context(), requestTimeout)
		defer cancel()

		requestBody := &ValidateContractRequestBody{
			Participant: participant,
			Environment: environment,
			Contracts:   contracts,
		}

		response, err := validateContractClient.ValidateContract(ctx, requestBody)
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

		checkedSideLabel := participant + " local contract"
		writer := command.OutOrStdout()

		if !response.Deployable {
			if _, err := fmt.Fprint(writer, reports.FormatNotDeployableReport(paint.For(writer), checkedSideLabel, participant, environment, response.Results)); err != nil {
				return err
			}

			return reports.ErrSilent
		}

		if _, err := fmt.Fprintln(writer, reports.FormatDeployableLine(paint.For(writer), checkedSideLabel, environment)); err != nil {
			return err
		}

		return nil
	}

	command := &cobra.Command{
		Use:   "validate [file...]",
		Short: "Validate contract YAML files against an environment without publishing them",
		Args:  cobra.MinimumNArgs(1),
		RunE:  commandHandler,
	}

	command.Flags().String("participant", "", "Participant name (required)")
	command.Flags().String("environment", "", "Target environment name (required)")
	cobra.CheckErr(command.MarkFlagRequired("participant"))
	cobra.CheckErr(command.MarkFlagRequired("environment"))

	return command
}
