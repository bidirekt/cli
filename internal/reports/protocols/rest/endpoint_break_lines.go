package rest

import (
	"fmt"
	"strings"

	"github.com/bidirekt/cli/internal/reports"
)

func formatBreakLine(interaction checkedInteraction, contractBreak reports.ContractBreak) reports.BreakLine {
	me, other := interaction.participant, interaction.counterpart
	resource := interaction.method + " " + interaction.endpoint
	reason, details := contractBreak.Reason, contractBreak.Details
	asConsumer, asProvider := contractBreak.Role == "consumer", contractBreak.Role == "provider"
	onRequest, onResponse := interaction.isRequest, !interaction.isRequest

	switch {
	case reason == "provider_resource_not_found" && asConsumer:
		return reports.BreakLine{
			Text:  fmt.Sprintf("%s calls %s, but %s doesn't provide it", me, resource, other),
			Fixes: []string{me + ": stop calling it", other + ": publish it"},
		}
	case reason == "provider_resource_interaction_not_found" && asConsumer && onRequest:
		return reports.BreakLine{
			Text:  fmt.Sprintf("%s sends a request body, but %s declares none", me, other),
			Fixes: []string{me + ": stop sending it", other + ": accept a request body"},
		}
	case reason == "provider_resource_interaction_not_found" && asConsumer && onResponse && details["providerStatuses"] == "":
		return reports.BreakLine{
			Text:  fmt.Sprintf("%s reads response %s, but %s declares no response", me, interaction.status, other),
			Fixes: []string{me + ": stop reading it", other + ": answer " + interaction.status},
		}
	case reason == "provider_resource_interaction_not_found" && asConsumer && onResponse:
		providerStatuses := orList(details["providerStatuses"])
		return reports.BreakLine{
			Text:  fmt.Sprintf("%s reads response %s, but %s answers only %s", me, interaction.status, other, providerStatuses),
			Fixes: []string{me + ": read response " + providerStatuses, other + ": answer " + interaction.status},
		}
	case reason == "provider_resource_not_deployed_in_environment" && asConsumer:
		return reports.BreakLine{
			Text:  fmt.Sprintf("%s calls %s, but %s is not deployed in %s", me, resource, other, interaction.environment),
			Fixes: []string{fmt.Sprintf("%s: deploy it to %s first", other, interaction.environment)},
		}
	case reason == "provider_resource_removed_but_still_consumed" && asProvider:
		return reports.BreakLine{
			Text:  fmt.Sprintf("%s removed %s, but %s still calls it", me, resource, other),
			Fixes: []string{fmt.Sprintf("%s: keep it until %s stops calling it", me, other), other + ": stop calling it"},
		}
	case reason == "provider_resource_interaction_removed_but_still_consumed" && asProvider && onRequest:
		return reports.BreakLine{
			Text:  fmt.Sprintf("%s dropped the request body of %s, but %s still sends it", me, resource, other),
			Fixes: []string{fmt.Sprintf("%s: keep accepting it until %s stops sending it", me, other), other + ": stop sending it"},
		}
	case reason == "provider_resource_interaction_removed_but_still_consumed" && asProvider && onResponse && details["providerStatuses"] == "":
		return reports.BreakLine{
			Text:  fmt.Sprintf("%s no longer answers %s, but %s still reads response %s", me, resource, other, interaction.status),
			Fixes: []string{fmt.Sprintf("%s: keep answering %s until %s stops reading it", me, interaction.status, other), other + ": stop reading it"},
		}
	case reason == "provider_resource_interaction_removed_but_still_consumed" && asProvider && onResponse:
		providerStatuses := orList(details["providerStatuses"])
		return reports.BreakLine{
			Text:  fmt.Sprintf("%s now answers %s only with %s, but %s still reads response %s", me, resource, providerStatuses, other, interaction.status),
			Fixes: []string{fmt.Sprintf("%s: keep answering %s until %s stops reading it", me, interaction.status, other), other + ": read response " + providerStatuses},
		}
	default:
		return reports.BreakLine{Text: reports.FallbackBreakLine(reason, details)}
	}
}

func orList(commaList string) string {
	last := strings.LastIndex(commaList, ", ")
	if last < 0 {
		return commaList
	}
	return commaList[:last] + " or " + commaList[last+len(", "):]
}
