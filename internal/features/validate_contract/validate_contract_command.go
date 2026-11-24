package validate_contract

import (
	"context"
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

		contracts, err := contractfiles.ToContractFiles(args)
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
			return reports.WriteValidationFailed(command.ErrOrStderr(), err)
		}

		if participant == "" {
			writer := command.OutOrStdout()
			_, err := fmt.Fprint(writer, reports.FormatSyntaxValidReport(paint.For(writer)))
			return err
		}

		checkedSideLabel := participant + " local contract"

		return reports.WriteVerdict(command.OutOrStdout(), checkedSideLabel, participant, response.Environment, response.Deployable, response.Results)
	}

	command := &cobra.Command{
		Use:   "validate [file or folder...]",
		Short: "Validate contract YAML files or folders, and compare them with an environment, without publishing them",
		Args:  cobra.MatchAll(cobra.MinimumNArgs(1), rejectEmptyFlags("participant", "environment")),
		RunE:  commandHandler,
	}

	command.Flags().String("participant", "", "Participant name (with --environment; without both, only the files are checked)")
	command.Flags().String("environment", "", "Environment to compare with (with --participant; without both, only the files are checked)")
	command.MarkFlagsRequiredTogether("participant", "environment")

	return command
}

func rejectEmptyFlags(names ...string) cobra.PositionalArgs {
	return func(command *cobra.Command, _ []string) error {
		for _, name := range names {
			flag := command.Flags().Lookup(name)
			if flag.Changed && flag.Value.String() == "" {
				return fmt.Errorf("--%s cannot be empty", name)
			}
		}

		return nil
	}
}
