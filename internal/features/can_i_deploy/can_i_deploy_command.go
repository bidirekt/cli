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
			if _, err := fmt.Fprint(writer, formatNotDeployableReport(paint.For(writer), participant, version, environment, resp.Results)); err != nil {
				return err
			}
			return ErrSilent
		}

		writer := command.OutOrStdout()
		if _, err := fmt.Fprintln(writer, formatDeployableLine(paint.For(writer), participant, version, environment)); err != nil {
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

func formatDeployableLine(brush paint.Brush, participant, version, environment string) string {
	return brush.Green(participant + " " + version + " can be deployed to " + environment)
}

func formatNotDeployableReport(brush paint.Brush, participant, version, environment string, results map[string]CanIDeployResult) string {
	var report strings.Builder
	report.WriteString(brush.Red(participant+" "+version+" cannot be deployed to "+environment) + "\n")

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
			fmt.Fprintf(&report, " (%s, deployed)", *result.ParticipantVersion)
		}
		report.WriteString(":\n")

		endpoints := sortedKeys(result.Endpoints)
		for _, endpoint := range endpoints {
			methods := sortedKeys(result.Endpoints[endpoint])
			for _, method := range methods {
				fmt.Fprintf(&report, "  %s %s\n", strings.ToUpper(method), endpoint)

				interactions := result.Endpoints[endpoint][method]
				for _, interaction := range sortedInteractions(interactions) {
					checked := checkedInteraction{
						participant: participant,
						counterpart: name,
						environment: environment,
						method:      strings.ToUpper(method),
						endpoint:    endpoint,
						isRequest:   interaction == "request",
					}

					if checked.isRequest {
						report.WriteString("    request:\n")
					} else {
						fmt.Fprintf(&report, "    response %s:\n", interaction)
					}

					for _, contractBreak := range interactions[interaction] {
						fmt.Fprintf(&report, "      - %s\n", formatBreakLine(checked, contractBreak))
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

type checkedInteraction struct {
	participant string
	counterpart string
	environment string
	method      string
	endpoint    string
	isRequest   bool
}

func formatBreakLine(interaction checkedInteraction, contractBreak ContractBreak) string {
	me, other := interaction.participant, interaction.counterpart
	resource := interaction.method + " " + interaction.endpoint
	reason, details := contractBreak.Reason, contractBreak.Details
	property := details["property"]
	consumerType, providerType := details["consumerPropertyType"], details["providerPropertyType"]
	asConsumer, asProvider := contractBreak.Role == "consumer", contractBreak.Role == "provider"
	onRequest, onResponse := interaction.isRequest, !interaction.isRequest

	switch {
	case reason == "property_missing_in_provider" && asConsumer && onResponse:
		return fmt.Sprintf("%s reads %q, but %s doesn't provide it → stop reading it, or mark it optional", me, property, other)
	case reason == "property_missing_in_provider" && asProvider && onResponse:
		return fmt.Sprintf("%s doesn't provide %q, but %s reads it → keep providing it", me, property, other)
	case reason == "property_optional_in_provider_required_in_consumer" && asConsumer && onResponse:
		return fmt.Sprintf("%s requires %q, but %s only sometimes provides it → mark it optional", me, property, other)
	case reason == "property_optional_in_provider_required_in_consumer" && asProvider && onResponse:
		return fmt.Sprintf("%s provides %q only sometimes, but %s requires it → keep it required", me, property, other)
	case reason == "property_missing_in_consumer" && asConsumer && onRequest:
		return fmt.Sprintf("%s doesn't send %q, but %s requires it → send it", me, property, other)
	case reason == "property_missing_in_consumer" && asProvider && onRequest:
		return fmt.Sprintf("%s requires %q, but %s doesn't send it → make it optional", me, property, other)
	case reason == "property_optional_in_consumer_required_in_provider" && asConsumer && onRequest:
		return fmt.Sprintf("%s sends %q only sometimes, but %s requires it → always send it", me, property, other)
	case reason == "property_optional_in_consumer_required_in_provider" && asProvider && onRequest:
		return fmt.Sprintf("%s requires %q, but %s sends it only sometimes → make it optional", me, property, other)
	case reason == "property_type_mismatch" && asConsumer && onResponse:
		return fmt.Sprintf("%s reads %q as %s, but %s provides %s → read it as %s", me, property, consumerType, other, providerType, providerType)
	case reason == "property_type_mismatch" && asProvider && onResponse:
		return fmt.Sprintf("%s provides %q as %s, but %s reads %s → provide %s", me, property, providerType, other, consumerType, consumerType)
	case reason == "property_type_mismatch" && asConsumer && onRequest:
		return fmt.Sprintf("%s sends %q as %s, but %s expects %s → send %s", me, property, consumerType, other, providerType, providerType)
	case reason == "property_type_mismatch" && asProvider && onRequest:
		return fmt.Sprintf("%s expects %q as %s, but %s sends %s → accept %s", me, property, providerType, other, consumerType, consumerType)
	case reason == "provider_resource_not_found" && asConsumer:
		return fmt.Sprintf("%s calls %s, but %s doesn't provide it → stop calling it, or wait until %s publishes it", me, resource, other, other)
	case reason == "provider_resource_not_deployed_in_environment" && asConsumer:
		deployedIn := ""
		if environments, ok := details["deployedEnvironments"]; ok {
			deployedIn = fmt.Sprintf(" (deployed in: %s)", environments)
		}
		return fmt.Sprintf("%s calls %s, but %s is not deployed in %s%s → deploy %s first", me, resource, other, interaction.environment, deployedIn, other)
	case reason == "provider_resource_removed_but_still_consumed" && asProvider:
		return fmt.Sprintf("%s removed %s, but %s still calls it → keep it until %s stops calling it", me, resource, other, other)
	default:
		return fallbackBreakLine(reason, details)
	}
}

// fallbackBreakLine renders a break verbatim with its details when no template matches its
// reason, role and interaction, so the CLI never swallows a break.
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
