package graphql

import (
	"testing"

	"github.com/bidirekt/cli/internal/reports"
	"github.com/stretchr/testify/assert"
)

func TestFormatBreakLine(t *testing.T) {
	webCallsPetOnApi := checkedOperation{participant: "petstore_web", counterpart: "petstore_api", environment: "production", operation: "query pet"}
	webCallsPetOnGraph := checkedOperation{participant: "petstore_web", counterpart: "petstore_graph", environment: "production", operation: "query pet"}
	apiServesPetToWeb := checkedOperation{participant: "petstore_api", counterpart: "petstore_web", environment: "production", operation: "query pet"}
	reviewsServesPetToWeb := checkedOperation{participant: "petstore_reviews", counterpart: "petstore_web", environment: "production", operation: "query pet"}

	tests := []struct {
		name          string
		operation     checkedOperation
		contractBreak reports.ContractBreak
		line          reports.BreakLine
	}{
		{
			name:          "provider_resource_not_found as consumer",
			operation:     webCallsPetOnApi,
			contractBreak: reports.ContractBreak{Reason: "provider_resource_not_found", Role: "consumer"},
			line: reports.BreakLine{
				Text:  "petstore_web calls query pet, but petstore_api doesn't provide it",
				Fixes: []string{"petstore_web: stop calling it", "petstore_api: publish it"},
			},
		},
		{
			name:          "provider_resource_removed_but_still_consumed as provider",
			operation:     apiServesPetToWeb,
			contractBreak: reports.ContractBreak{Reason: "provider_resource_removed_but_still_consumed", Role: "provider"},
			line: reports.BreakLine{
				Text:  "petstore_api removed query pet, but petstore_web still calls it",
				Fixes: []string{"petstore_api: keep it until petstore_web stops calling it", "petstore_web: stop calling it"},
			},
		},
		{
			name:      "provider_resource_removed_but_still_consumed through a graph names the subgraph via the graph",
			operation: reviewsServesPetToWeb,
			contractBreak: reports.ContractBreak{Reason: "provider_resource_removed_but_still_consumed", Role: "provider", Details: map[string]string{
				"graph": "petstore_graph",
			}},
			line: reports.BreakLine{
				Text:  "petstore_reviews (via petstore_graph) removed query pet, but petstore_web still calls it",
				Fixes: []string{"petstore_reviews: keep it until petstore_web stops calling it", "petstore_web: stop calling it"},
			},
		},
		{
			name:          "provider_resource_not_deployed_in_environment of a direct provider",
			operation:     webCallsPetOnApi,
			contractBreak: reports.ContractBreak{Reason: "provider_resource_not_deployed_in_environment", Role: "consumer"},
			line: reports.BreakLine{
				Text:  "petstore_web calls query pet, but petstore_api is not deployed in production",
				Fixes: []string{"petstore_api: deploy it to production first"},
			},
		},
		{
			name:      "provider_resource_not_deployed_in_environment of a graph asks for a subgraph",
			operation: webCallsPetOnGraph,
			contractBreak: reports.ContractBreak{Reason: "provider_resource_not_deployed_in_environment", Role: "consumer", Details: map[string]string{
				"graph": "petstore_graph",
			}},
			line: reports.BreakLine{
				Text:  "petstore_web calls query pet, but petstore_graph is not deployed in production",
				Fixes: []string{"petstore_graph: deploy a subgraph of it to production first"},
			},
		},
		{
			name:      "provider_graph_not_composable as consumer",
			operation: webCallsPetOnGraph,
			contractBreak: reports.ContractBreak{Reason: "provider_graph_not_composable", Role: "consumer", Details: map[string]string{
				"graph": "petstore_graph",
				"cause": "petstore_api 1.4.0 and petstore_orders 1.0.0 both resolve Money.amount without @shareable",
			}},
			line: reports.BreakLine{
				Text:  "petstore_web calls query pet, but petstore_graph can't be built in production: petstore_api 1.4.0 and petstore_orders 1.0.0 both resolve Money.amount without @shareable",
				Fixes: []string{"petstore_graph subgraphs: make it compose (see /graphs/petstore_graph)"},
			},
		},
		{
			name:      "provider_graph_not_composable as the subgraph under check",
			operation: reviewsServesPetToWeb,
			contractBreak: reports.ContractBreak{Reason: "provider_graph_not_composable", Role: "provider", Details: map[string]string{
				"graph": "petstore_graph",
				"cause": "petstore_api 1.4.0 resolves Pet.age as Int, but petstore_reviews 2.1.0 as String",
			}},
			line: reports.BreakLine{
				Text:  "petstore_reviews (via petstore_graph) makes petstore_graph fail to compose in production: petstore_api 1.4.0 resolves Pet.age as Int, but petstore_reviews 2.1.0 as String, and petstore_web calls query pet",
				Fixes: []string{"petstore_reviews: make petstore_graph compose (see /graphs/petstore_graph)"},
			},
		},
		{
			name:          "an unknown reason falls back to the reason and its details",
			operation:     webCallsPetOnApi,
			contractBreak: reports.ContractBreak{Reason: "provider_operation_deprecated", Role: "consumer", Details: map[string]string{"since": "1.5.0"}},
			line:          reports.BreakLine{Text: "provider_operation_deprecated (since: 1.5.0)"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.line, formatBreakLine(test.operation, test.contractBreak))
		})
	}
}
