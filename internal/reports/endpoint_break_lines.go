package reports

import (
	"fmt"
	"sort"
	"strings"
)

type breakLine struct {
	text  string
	fixes []string
}

func writeBreakLine(report *strings.Builder, indent string, line breakLine) {
	fmt.Fprintf(report, "%s- %s\n", indent, line.text)
	for _, fix := range line.fixes {
		fmt.Fprintf(report, "%s  → %s\n", indent, fix)
	}
}

func formatBreakLine(interaction checkedInteraction, contractBreak ContractBreak) breakLine {
	me, other := interaction.participant, interaction.counterpart
	resource := interaction.method + " " + interaction.endpoint
	reason, details := contractBreak.Reason, contractBreak.Details
	asConsumer, asProvider := contractBreak.Role == "consumer", contractBreak.Role == "provider"
	onRequest, onResponse := interaction.isRequest, !interaction.isRequest

	switch {
	case reason == "provider_resource_not_found" && asConsumer:
		return breakLine{
			text:  fmt.Sprintf("%s calls %s, but %s doesn't provide it", me, resource, other),
			fixes: []string{me + ": stop calling it", other + ": publish it"},
		}
	case reason == "provider_resource_interaction_not_found" && asConsumer && onRequest:
		return breakLine{
			text:  fmt.Sprintf("%s sends a request body, but %s declares none", me, other),
			fixes: []string{me + ": stop sending it", other + ": accept a request body"},
		}
	case reason == "provider_resource_interaction_not_found" && asConsumer && onResponse && details["providerStatuses"] == "":
		return breakLine{
			text:  fmt.Sprintf("%s reads response %s, but %s declares no response", me, interaction.status, other),
			fixes: []string{me + ": stop reading it", other + ": answer " + interaction.status},
		}
	case reason == "provider_resource_interaction_not_found" && asConsumer && onResponse:
		providerStatuses := orList(details["providerStatuses"])
		return breakLine{
			text:  fmt.Sprintf("%s reads response %s, but %s answers only %s", me, interaction.status, other, providerStatuses),
			fixes: []string{me + ": read response " + providerStatuses, other + ": answer " + interaction.status},
		}
	case reason == "provider_resource_not_deployed_in_environment" && asConsumer:
		return breakLine{
			text:  fmt.Sprintf("%s calls %s, but %s is not deployed in %s", me, resource, other, interaction.environment),
			fixes: []string{fmt.Sprintf("%s: deploy it to %s first", other, interaction.environment)},
		}
	case reason == "provider_resource_removed_but_still_consumed" && asProvider:
		return breakLine{
			text:  fmt.Sprintf("%s removed %s, but %s still calls it", me, resource, other),
			fixes: []string{fmt.Sprintf("%s: keep it until %s stops calling it", me, other), other + ": stop calling it"},
		}
	case reason == "provider_resource_interaction_removed_but_still_consumed" && asProvider && onRequest:
		return breakLine{
			text:  fmt.Sprintf("%s dropped the request body of %s, but %s still sends it", me, resource, other),
			fixes: []string{fmt.Sprintf("%s: keep accepting it until %s stops sending it", me, other), other + ": stop sending it"},
		}
	case reason == "provider_resource_interaction_removed_but_still_consumed" && asProvider && onResponse && details["providerStatuses"] == "":
		return breakLine{
			text:  fmt.Sprintf("%s no longer answers %s, but %s still reads response %s", me, resource, other, interaction.status),
			fixes: []string{fmt.Sprintf("%s: keep answering %s until %s stops reading it", me, interaction.status, other), other + ": stop reading it"},
		}
	case reason == "provider_resource_interaction_removed_but_still_consumed" && asProvider && onResponse:
		providerStatuses := orList(details["providerStatuses"])
		return breakLine{
			text:  fmt.Sprintf("%s now answers %s only with %s, but %s still reads response %s", me, resource, providerStatuses, other, interaction.status),
			fixes: []string{fmt.Sprintf("%s: keep answering %s until %s stops reading it", me, interaction.status, other), other + ": read response " + providerStatuses},
		}
	default:
		return breakLine{text: fallbackBreakLine(reason, details)}
	}
}

func orList(commaList string) string {
	last := strings.LastIndex(commaList, ", ")
	if last < 0 {
		return commaList
	}
	return commaList[:last] + " or " + commaList[last+len(", "):]
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
