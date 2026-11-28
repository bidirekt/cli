package publish_contract

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

func NewPublishCommand(publishContractClient *PublishContractClient, modules []reports.Module) *cobra.Command {
	var extensions []string
	for _, module := range modules {
		extensions = append(extensions, module.Extensions...)
	}

	commandHandler := func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true
		command.SilenceErrors = true

		contracts, err := contractfiles.ToContractFiles(args, extensions)
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
			return reports.WriteValidationFailed(command.ErrOrStderr(), err, modules)
		}

		writer := command.OutOrStdout()
		brush := paint.For(writer)
		if _, err := fmt.Fprintln(writer, brush.Green(participant+" "+message)); err != nil {
			return err
		}

		return nil
	}

	command := &cobra.Command{
		Use:   "publish [contracts folder or file...]",
		Short: "Publish contract files to the broker, from the contracts/ folder when none is given",
		Long:  "Publish contract files to the broker, from the contracts/ folder when none is given.\n\n" + contractfiles.FileTypesHelp(extensions),
		RunE:  commandHandler,
	}

	command.Flags().String("participant", "", "Participant name (required)")
	command.Flags().String("version", "", "Contract version, e.g. a commit hash or semver tag (required)")
	_ = command.MarkFlagRequired("participant")
	_ = command.MarkFlagRequired("version")

	return command
}
