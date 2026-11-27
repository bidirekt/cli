package rest

import (
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/bidirekt/cli/internal/reports"
)

// splitRemovedBreak takes the removed-endpoint break out of a method's interactions: the broker sends it once per
// consumed interaction, but it is about the whole endpoint. Interactions left without breaks are dropped.
func splitRemovedBreak(interactions map[string][]reports.ContractBreak) (*reports.ContractBreak, map[string][]reports.ContractBreak) {
	var removed *reports.ContractBreak
	remaining := make(map[string][]reports.ContractBreak, len(interactions))
	for _, interaction := range sortedInteractions(interactions) {
		var kept []reports.ContractBreak
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

func splitFieldBlocks(checked checkedInteraction, interactions map[string][]reports.ContractBreak) ([]fieldBlock, map[string][]reports.ContractBreak) {
	var blocks []fieldBlock
	remaining := make(map[string][]reports.ContractBreak, len(interactions))
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
func sortedInteractions(interactions map[string][]reports.ContractBreak) []string {
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
