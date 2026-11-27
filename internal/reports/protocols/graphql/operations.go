package graphql

import (
	"maps"
	"slices"

	"github.com/bidirekt/cli/internal/reports"
)

type checkedOperation struct {
	participant string
	counterpart string
	environment string
	operation   string
	sendsIt     bool
}

// splitFieldBlocks keeps apart the breaks that are not about a field, in interaction order: the broker puts the
// breaks about the whole operation under "returns".
func splitFieldBlocks(checked checkedOperation, interactions map[string][]reports.ContractBreak) ([]fieldBlock, []reports.ContractBreak) {
	var blocks []fieldBlock
	var remaining []reports.ContractBreak
	for _, interaction := range slices.Sorted(maps.Keys(interactions)) {
		for _, contractBreak := range interactions[interaction] {
			if block, named := toFieldBlock(checked, interaction, contractBreak); named {
				blocks = append(blocks, block)
			} else {
				remaining = append(remaining, contractBreak)
			}
		}
	}
	return blocks, remaining
}
