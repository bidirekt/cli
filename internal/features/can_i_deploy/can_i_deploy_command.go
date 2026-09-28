package can_i_deploy

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bidirekt/cli/internal/paint"
	"github.com/spf13/cobra"
)

const requestTimeout = 30 * time.Second

var ErrSilent = errors.New("failure already reported")

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
			return ErrSilent
		}

		if !resp.Deployable {
			writer := command.OutOrStdout()
			if _, err := fmt.Fprint(writer, formatNotDeployableReport(paint.For(writer), participant, environment, resp.Results)); err != nil {
				return err
			}
			return ErrSilent
		}

		writer := command.OutOrStdout()
		brush := paint.For(writer)
		if _, err := fmt.Fprintln(writer, brush.Green(participant+" can be deployed to "+environment)); err != nil {
			return err
		}

		return nil
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

func formatNotDeployableReport(brush paint.Brush, participant, environment string, results map[string]CanIDeployResult) string {
	var report strings.Builder
	report.WriteString(brush.Red(participant+" cannot be deployed to "+environment) + "\n")

	counterparts := make([]string, 0, len(results))
	for name, result := range results {
		if !result.Deployable {
			counterparts = append(counterparts, name)
		}
	}
	sort.Strings(counterparts)

	for _, name := range counterparts {
		result := results[name]

		report.WriteString("\n" + name)
		if result.ParticipantVersion != nil {
			fmt.Fprintf(&report, " (%s)", *result.ParticipantVersion)
		}
		report.WriteString(":\n")

		endpoints := sortedKeys(result.Endpoints)
		for _, endpoint := range endpoints {
			methods := sortedKeys(result.Endpoints[endpoint])
			for _, method := range methods {
				fmt.Fprintf(&report, "  %s %s\n", strings.ToUpper(method), endpoint)

				interactions := result.Endpoints[endpoint][method]
				for _, interaction := range sortedInteractions(interactions) {
					if interaction == "request" {
						report.WriteString("    request:\n")
					} else {
						fmt.Fprintf(&report, "    response %s:\n", interaction)
					}

					for _, contractBreak := range interactions[interaction] {
						fmt.Fprintf(&report, "      - %s\n", formatBreakLine(environment, contractBreak))
					}
				}
			}
		}
	}

	return report.String()
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// sortedInteractions orders a method's interaction keys as "request" first, then the
// response status codes sorted.
func sortedInteractions(interactions map[string][]ContractBreak) []string {
	keys := sortedKeys(interactions)
	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i] == "request" && keys[j] != "request"
	})
	return keys
}

func formatBreakLine(environment string, contractBreak ContractBreak) string {
	details := contractBreak.Details

	switch contractBreak.Reason {
	case "provider_resource_not_deployed_in_environment":
		line := fmt.Sprintf("provider is not deployed in %q", environment)
		if deployedIn, ok := details["deployedEnvironments"]; ok {
			line += fmt.Sprintf(" (deployed in: %s)", deployedIn)
		}
		return line
	case "provider_resource_not_found":
		return "no matching resource in provider"
	case "provider_resource_removed_but_still_consumed":
		return "resource removed but still consumed"
	case "property_missing_in_provider":
		return fmt.Sprintf("property %q is missing in provider", details["property"])
	case "property_missing_in_consumer":
		return fmt.Sprintf("property %q is missing in consumer", details["property"])
	case "property_optional_in_provider_required_in_consumer":
		return fmt.Sprintf("property %q is optional in provider but required in consumer", details["property"])
	case "property_optional_in_consumer_required_in_provider":
		return fmt.Sprintf("property %q is optional in consumer but required in provider", details["property"])
	case "property_type_mismatch":
		return fmt.Sprintf("property %q type mismatch — consumer has %s, provider has %s",
			details["property"], details["consumerPropertyType"], details["providerPropertyType"])
	default:
		return fallbackBreakLine(contractBreak.Reason, details)
	}
}

// fallbackBreakLine renders an unknown reason code verbatim with its details, so the
// CLI never swallows a break it does not have a template for.
func fallbackBreakLine(reason string, details map[string]string) string {
	if len(details) == 0 {
		return reason
	}

	keys := make([]string, 0, len(details))
	for key := range details {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, key+": "+details[key])
	}

	return fmt.Sprintf("%s (%s)", reason, strings.Join(pairs, ", "))
}
