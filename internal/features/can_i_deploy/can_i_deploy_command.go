package can_i_deploy

import (
	"context"
	"fmt"
	"time"

	"github.com/bidirekt/cli/internal/paint"
	"github.com/bidirekt/cli/internal/reports"
	"github.com/spf13/cobra"
)

const requestTimeout = 30 * time.Second

func NewCanIDeployCommand(client *CanIDeployClient) *cobra.Command {
	commandHandler := func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true
		command.SilenceErrors = true

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

		requestBody := &CanIDeployRequestBody{
			Participant: participant,
			Version:     version,
			Environment: environment,
		}

		resp, err := client.Check(ctx, requestBody)
		if err != nil {
			errWriter := command.ErrOrStderr()
			if _, err := fmt.Fprintln(errWriter, paint.For(errWriter).Red(err.Error())); err != nil {
				return err
			}
			return reports.ErrSilent
		}

		checkedSideLabel := participant + " " + version

		return reports.WriteVerdict(command.OutOrStdout(), checkedSideLabel, participant, environment, resp.Deployable, resp.Results)
	}

	command := &cobra.Command{
		Use:   "can-i-deploy [participant]",
		Short: "Check whether a participant version can be deployed to an environment",
		Args:  cobra.ExactArgs(1),
		RunE:  commandHandler,
	}

	command.Flags().String("version", "", "Version to check, e.g. a commit hash or semver tag (required)")
	command.Flags().String("environment", "", "Target environment name (required)")
	cobra.CheckErr(command.MarkFlagRequired("version"))
	cobra.CheckErr(command.MarkFlagRequired("environment"))

	return command
}
