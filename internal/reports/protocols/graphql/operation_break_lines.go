package graphql

import (
	"fmt"

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
		meInPhrase = fmt.Sprintf("%s (via %s)", me, graph)
	}

	switch {
	case reason == "provider_resource_not_found" && asConsumer:
		return reports.BreakLine{
			Text:  fmt.Sprintf("%s calls %s, but %s doesn't provide it", me, operation, other),
			Fixes: []string{me + ": stop calling it", other + ": publish it"},
		}
	case reason == "provider_resource_not_deployed_in_environment" && asConsumer && graph != "":
		return reports.BreakLine{
			Text:  fmt.Sprintf("%s calls %s, but %s is not deployed in %s", me, operation, graph, environment),
			Fixes: []string{fmt.Sprintf("%s: deploy a subgraph of it to %s first", graph, environment)},
		}
	case reason == "provider_resource_not_deployed_in_environment" && asConsumer:
		return reports.BreakLine{
			Text:  fmt.Sprintf("%s calls %s, but %s is not deployed in %s", me, operation, other, environment),
			Fixes: []string{fmt.Sprintf("%s: deploy it to %s first", other, environment)},
		}
	case reason == "provider_resource_removed_but_still_consumed" && asProvider:
		return reports.BreakLine{
			Text:  fmt.Sprintf("%s removed %s, but %s still calls it", meInPhrase, operation, other),
			Fixes: []string{fmt.Sprintf("%s: keep it until %s stops calling it", me, other), other + ": stop calling it"},
		}
	case reason == "provider_graph_not_composable" && asConsumer:
		return reports.BreakLine{
			Text:  fmt.Sprintf("%s calls %s, but %s can't be built in %s: %s", me, operation, graph, environment, details["cause"]),
			Fixes: []string{fmt.Sprintf("%s subgraphs: make it compose (see /graphs/%s)", graph, graph)},
		}
	case reason == "provider_graph_not_composable" && asProvider:
		return reports.BreakLine{
			Text:  fmt.Sprintf("%s makes %s fail to compose in %s: %s, and %s calls %s", meInPhrase, graph, environment, details["cause"], other, operation),
			Fixes: []string{fmt.Sprintf("%s: make %s compose (see /graphs/%s)", me, graph, graph)},
		}
	default:
		return reports.BreakLine{Text: reports.FallbackBreakLine(reason, details)}
	}
}
