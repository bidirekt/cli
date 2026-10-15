package reports

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/bidirekt/cli/internal/paint"
)

// Endpoints nest as endpoint -> method -> interaction -> breaks, where interaction is
// "request" or a response status code ("200").
type CanIDeployResult struct {
	Deployable         bool                                             `json:"deployable"`
	ParticipantVersion *string                                          `json:"participantVersion"`
	Endpoints          map[string]map[string]map[string][]ContractBreak `json:"endpoints"`
}

type ContractBreak struct {
	Reason  string            `json:"reason"`
	Role    string            `json:"role"`
	Details map[string]string `json:"details"`
}

func FormatDeployableReport(brush paint.Brush, checkedSideLabel, environment string, results map[string]CanIDeployResult) string {
	report := brush.Green(checkedSideLabel+" can be deployed to "+environment) + "\n"
	if len(results) == 0 {
		return report + "nothing deployed in " + environment + " to compare with\n"
	}
	return report + "compared with: " + strings.Join(sortedKeys(results), ", ") + "\n"
}

func FormatNotDeployableReport(brush paint.Brush, checkedSideLabel, participant, environment string, results map[string]CanIDeployResult) string {
	var report strings.Builder
	report.WriteString(brush.Red(checkedSideLabel+" cannot be deployed to "+environment) + "\n")

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

				checked := checkedInteraction{
					participant: participant,
					counterpart: name,
					environment: environment,
					method:      strings.ToUpper(method),
					endpoint:    endpoint,
				}

				removed, interactions := splitRemovedBreak(result.Endpoints[endpoint][method])
				if removed != nil {
					writeBreakLine(&report, "    ", formatBreakLine(checked, *removed))
				}

				for _, interaction := range sortedInteractions(interactions) {
					checked.isRequest = interaction == "request"

					if checked.isRequest {
						checked.status = ""
						report.WriteString("    request:\n")
					} else {
						checked.status = interaction
						fmt.Fprintf(&report, "    response %s:\n", interaction)
					}

					for _, contractBreak := range interactions[interaction] {
						writeBreakLine(&report, "      ", formatBreakLine(checked, contractBreak))
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

// splitRemovedBreak takes the removed-endpoint break out of a method's interactions: the broker sends it once per
// consumed interaction, but it is about the whole endpoint. Interactions left without breaks are dropped.
func splitRemovedBreak(interactions map[string][]ContractBreak) (*ContractBreak, map[string][]ContractBreak) {
	var removed *ContractBreak
	remaining := make(map[string][]ContractBreak, len(interactions))
	for _, interaction := range sortedInteractions(interactions) {
		var kept []ContractBreak
		for _, contractBreak := range interactions[interaction] {
			if contractBreak.Reason != "provider_resource_removed_but_still_consumed" {
				kept = append(kept, contractBreak)
			} else if removed == nil {
				removed = &contractBreak
			}
		}
		if len(kept) > 0 {
			remaining[interaction] = kept
		}
	}
	return removed, remaining
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
	status      string
}

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
	consumerType, providerType := details["consumerPropertyType"], details["providerPropertyType"]
	asConsumer, asProvider := contractBreak.Role == "consumer", contractBreak.Role == "provider"
	consumer, provider := me, other
	if asProvider {
		consumer, provider = other, me
	}
	property := propertyBreak{
		consumer:          consumer,
		provider:          provider,
		consumerField:     details["consumerField"],
		providerField:     details["providerField"],
		path:              details["property"],
		consumerIsChecked: asConsumer,
		onRequest:         interaction.isRequest,
	}
	path := property.pathPhrase()
	onRequest, onResponse := interaction.isRequest, !interaction.isRequest

	switch {
	case !asConsumer && !asProvider:
		return breakLine{text: fallbackBreakLine(reason, details)}
	case reason == "property_missing_in_provider" && onResponse:
		return property.toBreakLine(propertyBreakPhrases{
			consumerChecked:        fmt.Sprintf("%s reads %s, but %s doesn't provide it", consumer, path, provider),
			providerChecked:        fmt.Sprintf("%s doesn't provide %s, but %s reads it", provider, path, consumer),
			consumerFix:            "stop reading it, or handle its absence in the code and mark it optional",
			providerFix:            "provide it",
			readerFixMarksOptional: true,
		})
	case reason == "property_optional_in_provider_required_in_consumer" && onResponse:
		return property.toBreakLine(propertyBreakPhrases{
			consumerChecked:        fmt.Sprintf("%s requires %s, but %s only sometimes provides it", consumer, path, provider),
			providerChecked:        fmt.Sprintf("%s provides %s only sometimes, but %s requires it", provider, path, consumer),
			consumerFix:            "handle its absence in the code, then mark it optional",
			providerFix:            "always provide it",
			readerFixMarksOptional: true,
		})
	case reason == "property_type_mismatch" && onResponse:
		return property.toBreakLine(propertyBreakPhrases{
			consumerChecked: fmt.Sprintf("%s reads %s as %s, but %s provides %s", consumer, path, consumerType, provider, providerType),
			providerChecked: fmt.Sprintf("%s provides %s as %s, but %s reads %s", provider, path, providerType, consumer, consumerType),
			consumerFix:     "read it as " + providerType,
			providerFix:     "provide it as " + consumerType,
		})
	case reason == "property_nullable_in_provider_not_nullable_in_consumer" && onResponse:
		return property.toBreakLine(propertyBreakPhrases{
			consumerChecked: fmt.Sprintf("%s doesn't accept null at %s, but %s may provide null", consumer, path, provider),
			providerChecked: fmt.Sprintf("%s may provide null at %s, but %s doesn't accept it", provider, path, consumer),
			consumerFix:     "handle null in the code, then mark it nullable",
			providerFix:     "stop providing null",
		})
	case reason == "property_missing_in_consumer" && onRequest:
		return property.toBreakLine(propertyBreakPhrases{
			consumerChecked:        fmt.Sprintf("%s doesn't send %s, but %s requires it", consumer, path, provider),
			providerChecked:        fmt.Sprintf("%s requires %s, but %s doesn't send it", provider, path, consumer),
			consumerFix:            "send it",
			providerFix:            "stop requiring it, then mark it optional",
			readerFixMarksOptional: true,
		})
	case reason == "property_optional_in_consumer_required_in_provider" && onRequest:
		return property.toBreakLine(propertyBreakPhrases{
			consumerChecked:        fmt.Sprintf("%s sends %s only sometimes, but %s requires it", consumer, path, provider),
			providerChecked:        fmt.Sprintf("%s requires %s, but %s sends it only sometimes", provider, path, consumer),
			consumerFix:            "always send it",
			providerFix:            "handle its absence in the code, then mark it optional",
			readerFixMarksOptional: true,
		})
	case reason == "property_type_mismatch" && onRequest:
		return property.toBreakLine(propertyBreakPhrases{
			consumerChecked: fmt.Sprintf("%s sends %s as %s, but %s expects %s", consumer, path, consumerType, provider, providerType),
			providerChecked: fmt.Sprintf("%s expects %s as %s, but %s sends %s", provider, path, providerType, consumer, consumerType),
			consumerFix:     "send " + providerType,
			providerFix:     "accept " + consumerType,
		})
	case reason == "property_nullable_in_consumer_not_nullable_in_provider" && onRequest:
		return property.toBreakLine(propertyBreakPhrases{
			consumerChecked: fmt.Sprintf("%s may send null at %s, but %s doesn't accept it", consumer, path, provider),
			providerChecked: fmt.Sprintf("%s doesn't accept null at %s, but %s may send it", provider, path, consumer),
			consumerFix:     "stop sending null",
			providerFix:     "handle null in the code, then mark it nullable",
		})
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
		deployedIn := ""
		if environments, ok := details["deployedEnvironments"]; ok {
			deployedIn = fmt.Sprintf(" (deployed in: %s)", environments)
		}
		return breakLine{
			text:  fmt.Sprintf("%s calls %s, but %s is not deployed in %s%s", me, resource, other, interaction.environment, deployedIn),
			fixes: []string{fmt.Sprintf("%s: deploy it to %s first", other, interaction.environment)},
		}
	case reason == "provider_resource_removed_but_still_consumed" && asProvider:
		return breakLine{
			text:  fmt.Sprintf("%s removed %s, but %s still calls it", me, resource, other),
			fixes: []string{fmt.Sprintf("%s: keep it until %s stops calling it", me, other), other + ": stop calling it"},
		}
	default:
		return breakLine{text: fallbackBreakLine(reason, details)}
	}
}

type propertyBreak struct {
	consumer, provider           string
	consumerField, providerField string
	path                         string
	consumerIsChecked            bool
	onRequest                    bool
}

type propertyBreakPhrases struct {
	consumerChecked, providerChecked string
	consumerFix, providerFix         string
	readerFixMarksOptional           bool
}

func (this propertyBreak) pathPhrase() string {
	if this.consumerIsChecked {
		return propertyPathPhrase(this.path, this.consumerField)
	}
	return propertyPathPhrase(this.path, this.providerField)
}

func (this propertyBreak) toBreakLine(phrases propertyBreakPhrases) breakLine {
	consumerFix, consumerField := phrases.consumerFix, this.consumerField
	providerFix, providerField := phrases.providerFix, this.providerField

	// An array's items and a map's values aren't fields to mark optional: their reader reads the parent the way the
	// other side produces it, and the parent's path says where, so that fix names no field.
	isItemsOrValues := strings.HasSuffix(this.path, "[]") || strings.HasSuffix(this.path, "{}")
	if isItemsOrValues && phrases.readerFixMarksOptional {
		parent := propertyPathPhrase(this.path[:len(this.path)-len("[]")], "")
		if this.onRequest {
			providerFix, providerField = fmt.Sprintf("read %s the way %s sends it", parent, this.consumer), ""
		} else {
			consumerFix, consumerField = fmt.Sprintf("read %s the way %s provides it", parent, this.provider), ""
		}
	}

	consumerLine, providerLine := this.consumer+": "+consumerFix, this.provider+": "+providerFix
	if this.consumerIsChecked {
		return breakLine{text: phrases.consumerChecked, fixes: []string{consumerLine, providerLine + at(providerField)}}
	}
	return breakLine{text: phrases.providerChecked, fixes: []string{providerLine, consumerLine + at(consumerField)}}
}

func orList(commaList string) string {
	last := strings.LastIndex(commaList, ", ")
	if last < 0 {
		return commaList
	}
	return commaList[:last] + " or " + commaList[last+len(", "):]
}

func propertyPathPhrase(path, field string) string {
	phrase := strconv.Quote(path)
	if path == "$" {
		phrase = "the body"
	}
	if field != "" {
		phrase += " (" + field + ")"
	}
	return phrase
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
