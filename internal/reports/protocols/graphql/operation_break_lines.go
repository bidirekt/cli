package graphql

import (
	"fmt"
	"strings"

	"github.com/bidirekt/cli/internal/reports"
)

func formatBreakLine(checked checkedOperation, contractBreak reports.ContractBreak) reports.BreakLine {
	me, other := checked.participant, checked.counterpart
	operation, environment := checked.operation, checked.environment
	reason, details := contractBreak.Reason, contractBreak.Details
	graph := details["graph"]
	asConsumer, asProvider := contractBreak.Role == "consumer", contractBreak.Role == "provider"
	meInPhrase := me
	if asProvider && graph != "" {
		meInPhrase = viaGraph(me, graph)
	}
	toDeploy := "it"
	if graph != "" {
		toDeploy = "a subgraph of it"
	}

	switch {
	case reason == "provider_resource_not_found" && asConsumer:
		return reports.BreakLine{
			Text:  fmt.Sprintf("%s calls %s, but %s doesn't provide it", me, operation, other),
			Fixes: []string{me + ": stop calling it", other + ": publish it"},
		}
	case reason == "provider_resource_not_deployed_in_environment" && asConsumer:
		return reports.BreakLine{
			Text:  fmt.Sprintf("%s calls %s, but %s is not deployed in %s", me, operation, other, environment),
			Fixes: []string{fmt.Sprintf("%s: deploy %s to %s first", other, toDeploy, environment)},
		}
	case reason == "provider_resource_removed_but_still_consumed" && asProvider:
		return reports.BreakLine{
			Text:  fmt.Sprintf("%s removed %s, but %s still calls it", meInPhrase, operation, other),
			Fixes: []string{fmt.Sprintf("%s: keep it until %s stops calling it", me, other), other + ": stop calling it"},
		}
	case reason == "provider_graph_not_composable" && asConsumer:
		return reports.BreakLine{
			Text:   fmt.Sprintf("%s calls %s, but %s can't be built in %s", me, operation, graph, environment),
			Causes: compositionCauses(details),
			Fixes:  []string{graph + " subgraphs: make it compose"},
			Link:   deployedGraphLink(graph),
		}
	case reason == "provider_graph_not_composable" && asProvider:
		return reports.BreakLine{
			Text:   fmt.Sprintf("%s makes %s fail to compose in %s, and %s calls %s", meInPhrase, graph, environment, other, operation),
			Causes: compositionCauses(details),
			Fixes:  []string{fmt.Sprintf("%s: make %s compose", me, graph)},
			Link:   deployedGraphLink(graph),
		}
	default:
		return reports.BreakLine{Text: reports.FallbackBreakLine(reason, details)}
	}
}

func viaGraph(name, graph string) string {
	return fmt.Sprintf("%s (via %s)", name, graph)
}

// A broker older than causes sends only the first one.
func compositionCauses(details map[string]string) []string {
	if causes := details["causes"]; causes != "" {
		return strings.Split(causes, "\n")
	}

	return []string{details["cause"]}
}

func deployedGraphLink(graph string) string {
	return "deployed graph: /graphs/" + graph
}
