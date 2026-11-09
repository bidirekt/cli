package reports

import (
	"fmt"
	"maps"
	"slices"
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

		var blocks []fieldBlock
		endpoints := sortedKeys(result.Endpoints)
		for _, endpoint := range endpoints {
			methods := sortedKeys(result.Endpoints[endpoint])
			for _, method := range methods {
				checked := checkedInteraction{
					participant: participant,
					counterpart: name,
					environment: environment,
					method:      strings.ToUpper(method),
					endpoint:    endpoint,
				}

				removed, interactions := splitRemovedBreak(result.Endpoints[endpoint][method])
				methodBlocks, interactions := splitFieldBlocks(checked, interactions)
				blocks = append(blocks, methodBlocks...)
				if removed == nil && len(interactions) == 0 {
					continue
				}

				fmt.Fprintf(&report, "  %s %s\n", strings.ToUpper(method), endpoint)

				if removed != nil {
					writeBreakLine(&report, "    ", formatBreakLine(checked, *removed))
				}

				for _, interaction := range sortedInteractions(interactions) {
					checked.isRequest = interaction == "request"

					if checked.isRequest {
						checked.status = ""
					} else {
						checked.status = interaction
					}

					fmt.Fprintf(&report, "    %s:\n", interactionPhrase(interaction))
					for _, contractBreak := range interactions[interaction] {
						writeBreakLine(&report, "      ", formatBreakLine(checked, contractBreak))
					}
				}
			}
		}

		for _, block := range mergeFieldBlocks(blocks) {
			writeFieldBlock(&report, name, block)
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

func splitFieldBlocks(checked checkedInteraction, interactions map[string][]ContractBreak) ([]fieldBlock, map[string][]ContractBreak) {
	var blocks []fieldBlock
	remaining := make(map[string][]ContractBreak, len(interactions))
	for interaction, contractBreaks := range interactions {
		for _, contractBreak := range contractBreaks {
			if block, named := toFieldBlock(checked, interaction, contractBreak); named {
				blocks = append(blocks, block)
			} else {
				remaining[interaction] = append(remaining[interaction], contractBreak)
			}
		}
	}
	return blocks, remaining
}

// sortedInteractions orders a method's interaction keys as "request" first, then the
// response status codes sorted.
func sortedInteractions(interactions map[string][]ContractBreak) []string {
	return slices.SortedFunc(maps.Keys(interactions), compareInteractions)
}

func compareInteractions(a, b string) int {
	switch {
	case a == b:
		return 0
	case a == "request":
		return -1
	case b == "request":
		return 1
	default:
		return strings.Compare(a, b)
	}
}

func interactionPhrase(interaction string) string {
	if interaction == "request" {
		return "request"
	}
	return "response " + interaction
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

func propertyPathPhrase(path string) string {
	if path == "$" {
		return "the body"
	}
	return strconv.Quote(path)
}
