package rest

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/bidirekt/cli/internal/reports"
)

// Endpoints nest as endpoint -> method -> interaction -> breaks, where interaction is
// "request" or a response status code ("200").
type endpointsTree map[string]map[string]map[string][]reports.ContractBreak

type Plugin struct{}

func Module() reports.Module {
	return reports.Module{Report: Plugin{}}
}

func (this Plugin) ReportKey() string {
	return "endpoints"
}

func (this Plugin) SchemaKeys() map[string][]string {
	return nil
}

func (this Plugin) WriteTree(report *strings.Builder, counterpart reports.Counterpart, tree json.RawMessage) error {
	var endpoints endpointsTree
	if err := json.Unmarshal(tree, &endpoints); err != nil {
		return fmt.Errorf("cannot parse the endpoints of %s: %w", counterpart.Name, err)
	}

	var blocks []fieldBlock
	for _, endpoint := range slices.Sorted(maps.Keys(endpoints)) {
		for _, method := range slices.Sorted(maps.Keys(endpoints[endpoint])) {
			checked := checkedInteraction{
				participant: counterpart.CheckedParticipant,
				counterpart: counterpart.Name,
				environment: counterpart.Environment,
				method:      strings.ToUpper(method),
				endpoint:    endpoint,
			}

			removed, interactions := splitRemovedBreak(endpoints[endpoint][method])
			methodBlocks, interactions := splitFieldBlocks(checked, interactions)
			blocks = append(blocks, methodBlocks...)
			if removed == nil && len(interactions) == 0 {
				continue
			}

			fmt.Fprintf(report, "  %s %s\n", strings.ToUpper(method), endpoint)

			if removed != nil {
				reports.WriteBreakLine(report, "    ", formatBreakLine(checked, *removed))
			}

			for _, interaction := range sortedInteractions(interactions) {
				checked.isRequest = interaction == "request"

				if checked.isRequest {
					checked.status = ""
				} else {
					checked.status = interaction
				}

				fmt.Fprintf(report, "    %s:\n", interactionPhrase(interaction))
				for _, contractBreak := range interactions[interaction] {
					reports.WriteBreakLine(report, "      ", formatBreakLine(checked, contractBreak))
				}
			}
		}
	}

	for _, block := range mergeFieldBlocks(blocks) {
		writeFieldBlock(report, counterpart.Name, block)
	}

	return nil
}
