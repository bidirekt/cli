package graphql

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/bidirekt/cli/internal/reports"
)

// Operations nest as operation type -> field -> interaction -> breaks, where operation type is "query" or
// "mutation" and interaction is "arguments" or "returns".
type operationsTree map[string]map[string]map[string][]reports.ContractBreak

var schemaFileExtensions = []string{".graphql", ".graphqls", ".gql"}

type Plugin struct{}

func New() Plugin {
	return Plugin{}
}

func Module() reports.Module {
	return reports.Module{Report: New(), Extensions: schemaFileExtensions}
}

func (this Plugin) ReportKey() string {
	return "operations"
}

func (this Plugin) SchemaKeys() map[string][]string {
	return map[string][]string{"name": {`"type: scalar" (GraphQL contracts only)`}}
}

func (this Plugin) WriteTree(report *strings.Builder, counterpart reports.Counterpart, tree json.RawMessage) error {
	var operations operationsTree
	if err := json.Unmarshal(tree, &operations); err != nil {
		return fmt.Errorf("cannot parse the operations of %s: %w", counterpart.Name, err)
	}

	var blocks []fieldBlock
	for _, operationType := range slices.Sorted(maps.Keys(operations)) {
		for _, field := range slices.Sorted(maps.Keys(operations[operationType])) {
			checked := checkedOperation{
				participant: counterpart.CheckedParticipant,
				counterpart: counterpart.Name,
				environment: counterpart.Environment,
				operation:   operationType + " " + field,
			}

			fieldBlocks, resourceBreaks := splitFieldBlocks(checked, operations[operationType][field])
			blocks = append(blocks, fieldBlocks...)
			if len(resourceBreaks) == 0 {
				continue
			}

			fmt.Fprintf(report, "  %s\n", checked.operation)
			for _, contractBreak := range resourceBreaks {
				reports.WriteBreakLine(report, "    ", formatBreakLine(checked, contractBreak))
			}
		}
	}

	for _, block := range mergeFieldBlocks(blocks) {
		writeFieldBlock(report, counterpart.Name, block)
	}

	return nil
}
