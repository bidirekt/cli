package graphql_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/bidirekt/cli/internal/paint"
	"github.com/bidirekt/cli/internal/reports"
	"github.com/bidirekt/cli/internal/reports/protocols/graphql"
	"github.com/bidirekt/cli/internal/reports/protocols/rest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var graphqlModules = []reports.Module{{Report: graphql.New(), Extensions: []string{".graphql", ".graphqls"}}}

type operations map[string]map[string]map[string][]reports.ContractBreak

func operationsTree(t *testing.T, tree operations) map[string]json.RawMessage {
	t.Helper()

	raw, err := json.Marshal(tree)
	require.NoError(t, err)

	return map[string]json.RawMessage{"operations": raw}
}

func versionOf(version string) *string {
	return &version
}

func brokenApiOperations(t *testing.T) map[string]json.RawMessage {
	t.Helper()

	return operationsTree(t, operations{
		"query": {
			"pets": {
				"returns": {
					{Reason: "provider_resource_not_found", Role: "consumer"},
				},
			},
			"pet": {
				"arguments": {
					{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
						"consumerField": "PetArgs.owner", "property": "$.owner", "propertyType": "string",
						"consumerName": "petstore_web", "providerName": "petstore_api",
					}},
				},
				"returns": {
					{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
						"consumerField": "ReviewsArgs.first", "property": "$.reviews", "argument": "first", "propertyType": "integer",
						"consumerName": "petstore_web", "providerName": "petstore_api",
					}},
				},
			},
		},
	})
}

const brokenApiReport = `petstore_api (1.4.0, deployed):
  query pets
    - petstore_web calls query pets, but petstore_api doesn't provide it
      → petstore_web: stop calling it
      → petstore_api: publish it
  PetArgs.owner — petstore_web sends it, but petstore_api doesn't accept it
    → petstore_web: stop sending it
    → petstore_api: accept it
      · query pet arguments at "$.owner"
  ReviewsArgs.first — petstore_web sends it, but petstore_api doesn't accept it
    → petstore_web: stop sending it
    → petstore_api: accept it
      · query pet returns at "$.reviews" (argument "first")
`

func TestFormatNotDeployableReportWritesTheBreaksOfADirectGraphQLProvider(t *testing.T) {
	results := map[string]reports.CanIDeployResult{
		"petstore_api": {ParticipantVersion: versionOf("1.4.0"), Trees: brokenApiOperations(t)},
	}

	report, err := reports.FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 3.0.0", "petstore_web", "production", results, graphqlModules)

	require.NoError(t, err)
	assert.Equal(t, "petstore_web 3.0.0 cannot be deployed to production\n\n"+brokenApiReport, report)
}

func TestFormatNotDeployableReportWritesAGraphThatDoesNotCompose(t *testing.T) {
	results := map[string]reports.CanIDeployResult{
		"petstore_api": {ParticipantVersion: versionOf("1.4.0"), Trees: brokenApiOperations(t)},
		"petstore_graph": {
			Members: map[string]string{"petstore_reviews": "2.0.0", "petstore_api": "1.4.0", "petstore_orders": "1.0.0"},
			Trees: operationsTree(t, operations{
				"query": {
					"pet": {
						"returns": {
							{Reason: "provider_graph_not_composable", Role: "consumer", Details: map[string]string{
								"graph": "petstore_graph",
								"cause": "petstore_api 1.4.0 and petstore_orders 1.0.0 both resolve Money.amount without @shareable",
							}},
						},
					},
				},
			}),
		},
	}

	report, err := reports.FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 3.0.0", "petstore_web", "production", results, graphqlModules)

	require.NoError(t, err)
	assert.Equal(t, "petstore_web 3.0.0 cannot be deployed to production\n\n"+brokenApiReport+`
petstore_graph (petstore_api 1.4.0 + petstore_orders 1.0.0 + petstore_reviews 2.0.0, deployed):
  query pet
    - petstore_web calls query pet, but petstore_graph can't be built in production: petstore_api 1.4.0 and petstore_orders 1.0.0 both resolve Money.amount without @shareable
      → petstore_graph subgraphs: make it compose (see /graphs/petstore_graph)
`, report)
}

func TestFormatNotDeployableReportNamesTheGraphABreakOfTheCheckedSubgraphComesThrough(t *testing.T) {
	results := map[string]reports.CanIDeployResult{
		"petstore_web": {
			ParticipantVersion: versionOf("3.0.0"),
			Trees: operationsTree(t, operations{
				"query": {
					"pet": {
						"returns": {
							{Reason: "property_type_mismatch", Role: "provider", Details: map[string]string{
								"consumerField": "Pet.age", "providerField": "Pet.age", "property": "$.age", "graph": "petstore_graph",
								"consumerPropertyType": "string", "providerPropertyType": "integer",
								"consumerName": "petstore_web", "providerName": "petstore_reviews",
							}},
						},
					},
				},
			}),
		},
	}

	report, err := reports.FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_reviews 2.1.0", "petstore_reviews", "production", results, graphqlModules)

	require.NoError(t, err)
	assert.Equal(t, `petstore_reviews 2.1.0 cannot be deployed to production

petstore_web (3.0.0, deployed):
  Pet.age — petstore_reviews (via petstore_graph) provides it as integer, but petstore_web reads string
    → petstore_reviews: provide it as string
    → petstore_web: read it as integer there
      · query pet returns at "$.age" (petstore_web: Pet.age)
`, report)
}

func TestFormatNotDeployableReportTellsAGraphWithoutSubgraphsFromAProviderThatIsNotDeployed(t *testing.T) {
	results := map[string]reports.CanIDeployResult{
		"petstore_api": {
			Trees: operationsTree(t, operations{
				"query": {
					"pets": {
						"returns": {
							{Reason: "provider_resource_not_deployed_in_environment", Role: "consumer"},
						},
					},
				},
			}),
		},
		"petstore_graph": {
			Trees: operationsTree(t, operations{
				"query": {
					"pet": {
						"returns": {
							{Reason: "provider_resource_not_deployed_in_environment", Role: "consumer", Details: map[string]string{"graph": "petstore_graph"}},
						},
					},
				},
			}),
		},
	}

	report, err := reports.FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 3.0.0", "petstore_web", "production", results, graphqlModules)

	require.NoError(t, err)
	assert.Equal(t, `petstore_web 3.0.0 cannot be deployed to production

petstore_api:
  query pets
    - petstore_web calls query pets, but petstore_api is not deployed in production
      → petstore_api: deploy it to production first

petstore_graph:
  query pet
    - petstore_web calls query pet, but petstore_graph is not deployed in production
      → petstore_graph: deploy a subgraph of it to production first
`, report)
}

func TestFormatNotDeployableReportWritesTheCheckedSubgraphThatBreaksTheComposition(t *testing.T) {
	results := map[string]reports.CanIDeployResult{
		"petstore_web": {
			ParticipantVersion: versionOf("3.0.0"),
			Trees: operationsTree(t, operations{
				"query": {
					"pet": {
						"returns": {
							{Reason: "provider_graph_not_composable", Role: "provider", Details: map[string]string{
								"graph": "petstore_graph",
								"cause": "petstore_api 1.4.0 resolves Pet.age as Int, but petstore_reviews 2.1.0 as String",
							}},
						},
					},
				},
			}),
		},
	}

	report, err := reports.FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_reviews 2.1.0", "petstore_reviews", "production", results, graphqlModules)

	require.NoError(t, err)
	assert.Equal(t, `petstore_reviews 2.1.0 cannot be deployed to production

petstore_web (3.0.0, deployed):
  query pet
    - petstore_reviews (via petstore_graph) makes petstore_graph fail to compose in production: petstore_api 1.4.0 resolves Pet.age as Int, but petstore_reviews 2.1.0 as String, and petstore_web calls query pet
      → petstore_reviews: make petstore_graph compose (see /graphs/petstore_graph)
`, report)
}

func TestFormatNotDeployableReportWritesTheBreaksOfTheCheckedGraphQLProvider(t *testing.T) {
	results := map[string]reports.CanIDeployResult{
		"petstore_web": {
			ParticipantVersion: versionOf("3.0.0"),
			Trees: operationsTree(t, operations{
				"mutation": {
					"adoptPet": {
						"returns": {
							{Reason: "provider_resource_removed_but_still_consumed", Role: "provider"},
						},
					},
				},
				"query": {
					"pet": {
						"arguments": {
							{Reason: "property_missing_in_provider", Role: "provider", Details: map[string]string{
								"consumerField": "PetArgs.owner", "property": "$.owner", "propertyType": "string",
								"consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
						"returns": {
							{Reason: "property_missing_in_provider", Role: "provider", Details: map[string]string{
								"consumerField": "ReviewsArgs.filter", "property": "$.reviews", "argument": "filter.minStars", "propertyType": "integer",
								"consumerName": "petstore_web", "providerName": "petstore_api",
							}},
							{Reason: "property_nullable_in_provider_not_nullable_in_consumer", Role: "provider", Details: map[string]string{
								"consumerField": "Pet.name", "providerField": "Pet.name", "property": "$.name", "propertyType": "string",
								"consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
				},
			}),
		},
	}

	report, err := reports.FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_api 1.5.0", "petstore_api", "production", results, graphqlModules)

	require.NoError(t, err)
	assert.Equal(t, `petstore_api 1.5.0 cannot be deployed to production

petstore_web (3.0.0, deployed):
  mutation adoptPet
    - petstore_api removed mutation adoptPet, but petstore_web still calls it
      → petstore_api: keep it until petstore_web stops calling it
      → petstore_web: stop calling it
  Pet.name — petstore_api may provide null, but petstore_web doesn't accept it
    → petstore_api: stop providing null
    → petstore_web: handle null in the code, then mark it nullable there
      · query pet returns at "$.name" (petstore_web: Pet.name)
  PetArgs.owner — petstore_api doesn't accept it, but petstore_web sends it
    → petstore_api: accept it
    → petstore_web: stop sending it there
      · query pet arguments at "$.owner" (petstore_web: PetArgs.owner)
  ReviewsArgs.filter — petstore_api doesn't accept it, but petstore_web sends it
    → petstore_api: accept it
    → petstore_web: stop sending it there
      · query pet returns at "$.reviews" (argument "filter.minStars") (petstore_web: ReviewsArgs.filter)
`, report)
}

func TestFormatNotDeployableReportAsksTheConsumerToSendANewRequiredArgumentOfANestedField(t *testing.T) {
	results := map[string]reports.CanIDeployResult{
		"petstore_api": {
			ParticipantVersion: versionOf("1.5.0"),
			Trees: operationsTree(t, operations{
				"query": {
					"pet": {
						"returns": {
							{Reason: "property_missing_in_consumer", Role: "consumer", Details: map[string]string{
								"providerField": "Pet:reviews:arguments.first", "property": "$.reviews", "argument": "first", "propertyType": "integer",
								"consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
				},
			}),
		},
	}

	report, err := reports.FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 3.0.0", "petstore_web", "production", results, graphqlModules)

	require.NoError(t, err)
	assert.Equal(t, `petstore_web 3.0.0 cannot be deployed to production

petstore_api (1.5.0, deployed):
  Pet:reviews:arguments.first — petstore_web doesn't send it, but petstore_api requires it
    → petstore_web: send it
    → petstore_api: stop requiring it, then mark it optional there
      · query pet returns at "$.reviews" (argument "first") (petstore_api: Pet:reviews:arguments.first)
`, report)
}

func TestFormatNotDeployableReportWritesEachProtocolOfACounterpartAsItsModuleDoes(t *testing.T) {
	endpoints, err := json.Marshal(map[string]map[string]map[string][]reports.ContractBreak{
		"/pets/*": {
			"get": {
				"200": {
					{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
						"consumerField": "Pet.photoUrl", "property": "$.photoUrl", "propertyType": "string",
						"consumerName": "petstore_web", "providerName": "petstore_api",
					}},
				},
			},
		},
	})
	require.NoError(t, err)
	trees := operationsTree(t, operations{
		"query": {
			"pet": {
				"returns": {
					{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
						"consumerField": "Pet.photoUrl", "property": "$.photoUrl", "propertyType": "string",
						"consumerName": "petstore_web", "providerName": "petstore_api",
					}},
				},
			},
		},
	})
	trees["endpoints"] = endpoints
	results := map[string]reports.CanIDeployResult{
		"petstore_api": {ParticipantVersion: versionOf("1.4.0"), Trees: trees},
	}
	modules := []reports.Module{{Report: rest.New()}, {Report: graphql.New(), Extensions: []string{".graphql", ".graphqls"}}}

	report, err := reports.FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 3.0.0", "petstore_web", "production", results, modules)

	require.NoError(t, err)
	assert.Equal(t, `petstore_web 3.0.0 cannot be deployed to production

petstore_api (1.4.0, deployed):
  Pet.photoUrl — petstore_web reads it, but petstore_api doesn't provide it
    → petstore_web: stop reading it, or handle its absence in the code and mark it optional
    → petstore_api: provide it
      · GET /pets/* response 200 at "$.photoUrl"
  Pet.photoUrl — petstore_web reads it, but petstore_api doesn't provide it
    → petstore_web: stop reading it, or handle its absence in the code and mark it optional
    → petstore_api: provide it
      · query pet returns at "$.photoUrl"
`, report)
}

func TestWriteVerdictWritesAnOperationsTreeWithTheGraphQLModule(t *testing.T) {
	results := map[string]reports.CanIDeployResult{
		"petstore_api": {
			ParticipantVersion: versionOf("1.4.0"),
			Trees: operationsTree(t, operations{
				"query": {
					"pets": {
						"returns": {
							{Reason: "provider_resource_not_found", Role: "consumer"},
						},
					},
				},
			}),
		},
	}
	var writer bytes.Buffer

	err := reports.WriteVerdict(&writer, "petstore_web 3.0.0", "petstore_web", "production", false, results, graphqlModules)

	require.ErrorIs(t, err, reports.ErrSilent)
	assert.Equal(t, `petstore_web 3.0.0 cannot be deployed to production

petstore_api (1.4.0, deployed):
  query pets
    - petstore_web calls query pets, but petstore_api doesn't provide it
      → petstore_web: stop calling it
      → petstore_api: publish it
`, writer.String())
}

func TestFormatNotDeployableReportFailsOnAnOperationsTreeItCannotParse(t *testing.T) {
	results := map[string]reports.CanIDeployResult{
		"petstore_api": {Trees: map[string]json.RawMessage{"operations": json.RawMessage(`{"query": ["pet"]}`)}},
	}

	_, err := reports.FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 3.0.0", "petstore_web", "production", results, graphqlModules)

	require.ErrorContains(t, err, "cannot parse the operations of petstore_api")
}
