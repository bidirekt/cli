package publish_contract

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const requestTimeout = 30 * time.Second

var ErrSilent = errors.New("failure already reported")

var supportedFileExtensions = map[string]bool{".yaml": true, ".yml": true, ".json": true}

func NewPublishCommand(publishContractClient *PublishContractClient) *cobra.Command {

	commandHandler := func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true
		command.SilenceErrors = true

		contracts := make([]ContractFragment, 0, len(args))

		for _, filePath := range args {
			if !supportedFileExtensions[strings.ToLower(filepath.Ext(filePath))] {
				return fmt.Errorf("unsupported contract file extension: %q", filePath)
			}

			contractFileContent, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("read contract file: %w", err)
			}

			contracts = append(contracts, ContractFragment{
				Source:  filePath,
				Content: string(contractFileContent),
			})
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
			var validationFailed *ValidationFailedError
			if errors.As(err, &validationFailed) {
				if _, err := fmt.Fprint(command.ErrOrStderr(), formatValidationFailedReport(validationFailed.Message, validationFailed.Violations)); err != nil {
					return err
				}

				return ErrSilent
			}

			return err
		}

		if _, err := fmt.Fprintf(command.OutOrStdout(), "📜 %s %s\n", participant, message); err != nil {
			return err
		}

		return nil
	}

	command := &cobra.Command{
		Use:   "publish [file...]",
		Short: "Publish one or more contract YAML or JSON files to the broker",
		Args:  cobra.MinimumNArgs(1),
		RunE:  commandHandler,
	}

	command.Flags().String("participant", "", "Participant name (required)")
	command.Flags().String("version", "", "Contract version, e.g. a commit hash or semver tag (required)")
	_ = command.MarkFlagRequired("participant")
	_ = command.MarkFlagRequired("version")

	return command
}
