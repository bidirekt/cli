package graphql_test

import (
	"bytes"
	"testing"

	"github.com/bidirekt/cli/internal/paint"
	"github.com/bidirekt/cli/internal/reports"
	"github.com/bidirekt/cli/internal/reports/protocols/graphql"
	"github.com/stretchr/testify/assert"
)

func TestFormatValidationFailedReportRendersTheGraphQLCodes(t *testing.T) {
	modules := []reports.Module{graphql.Module()}

	report := reports.FormatValidationFailedReport(paint.For(&bytes.Buffer{}), "contract validation failed", []reports.Violation{
		{Code: "graph.without_schema", Path: "provides;graphql;graph", Source: "contracts/api.yaml", Details: map[string]string{"graph": "petstore_graph"}},
		{Code: "graph.required_by_federation", Source: "graph/schema.graphqls", Details: map[string]string{}},
	}, modules)

	assert.Equal(t, `contract validation failed
  - contracts/api.yaml: graph "petstore_graph" without a GraphQL schema at provides graphql, pass its .graphql, .graphqls or .gql in the same command
  - graph/schema.graphqls: the GraphQL schema links Apollo Federation, but no graph is declared, declare it in provides.graphql.graph
`, report)
}

func TestFormatViolationLine(t *testing.T) {
	modules := []reports.Module{graphql.Module()}

	for _, test := range []struct {
		name        string
		violation   reports.Violation
		headline    string
		explanation string
	}{
		{
			name: "schema.scalar_not_allowed",
			violation: reports.Violation{Code: "schema.scalar_not_allowed", Path: "schemas;Pet;properties;bornAt;type", Source: "contracts/web.yaml",
				Details: map[string]string{"protocol": "rest"}},
			headline: `type "scalar" is only allowed in GraphQL contracts at schemas Pet properties bornAt`,
		},
		{
			name: "schema.arguments_not_allowed",
			violation: reports.Violation{Code: "schema.arguments_not_allowed", Path: "schemas;Pet;properties;reviews;arguments", Source: "contracts/web.yaml",
				Details: map[string]string{"protocol": "rest"}},
			headline: `"arguments" is only allowed in GraphQL contracts at schemas Pet properties reviews`,
		},
		{
			name: "schema.arguments_not_object of a consumed field naming a string",
			violation: reports.Violation{Code: "schema.arguments_not_object", Path: "consumes;petstore_api;graphql;query;pet;arguments", Source: "petstore_web.yaml",
				Details: map[string]string{"schema": "PetId", "reason": "not_object", "type": "string"}},
			headline: `"arguments" must name an object schema at consumes petstore_api graphql query pet: PetId is string`,
		},
		{
			name: "schema.arguments_not_object of a consumed field naming a nullable object",
			violation: reports.Violation{Code: "schema.arguments_not_object", Path: "consumes;petstore_api;graphql;query;pet;arguments", Source: "petstore_web.yaml",
				Details: map[string]string{"schema": "MaybeArgs", "reason": "nullable"}},
			headline: `"arguments" must name a non-nullable schema at consumes petstore_api graphql query pet: MaybeArgs is nullable`,
		},
		{
			name: "schema.arguments_not_object of a property naming an integer",
			violation: reports.Violation{Code: "schema.arguments_not_object", Path: "schemas;MaybePet;properties;reviews;arguments", Source: "petstore_web.yaml",
				Details: map[string]string{"schema": "First", "reason": "not_object", "type": "integer"}},
			headline: `"arguments" must name an object schema at schemas MaybePet properties reviews: First is integer`,
		},
		{
			name: "schema.arguments_not_object of a property naming a nullable object",
			violation: reports.Violation{Code: "schema.arguments_not_object", Path: "schemas;MaybePet;properties;reviews;arguments", Source: "petstore_web.yaml",
				Details: map[string]string{"schema": "ReviewsArgs", "reason": "nullable"}},
			headline: `"arguments" must name a non-nullable schema at schemas MaybePet properties reviews: ReviewsArgs is nullable`,
		},
		{
			name: "schema.arguments_not_object with a reason this bidirekt doesn't know falls back to the code with its sorted details",
			violation: reports.Violation{Code: "schema.arguments_not_object", Path: "consumes;petstore_api;graphql;query;pet;arguments", Source: "petstore_web.yaml",
				Details: map[string]string{"schema": "PetId", "reason": "something_new"}},
			headline: `schema.arguments_not_object at consumes petstore_api graphql query pet arguments (reason: something_new, schema: PetId)`,
		},
		{
			name: "schema.scalar_without_name",
			violation: reports.Violation{Code: "schema.scalar_without_name", Path: "schemas;Pet;properties;bornAt", Source: "contracts/web.yaml",
				Details: map[string]string{}},
			headline: `scalar schema without name at schemas Pet properties bornAt`,
		},
		{
			name: "returns.required",
			violation: reports.Violation{Code: "returns.required", Path: "consumes;petstore_api;graphql;query;pet", Source: "contracts/web.yaml",
				Details: map[string]string{}},
			headline: `missing "returns" at consumes petstore_api graphql query pet`,
		},
		{
			name: "graph.without_schema",
			violation: reports.Violation{Code: "graph.without_schema", Path: "provides;graphql;graph", Source: "contracts/api.yaml",
				Details: map[string]string{"graph": "petstore_graph"}},
			headline:    `graph "petstore_graph" without a GraphQL schema at provides graphql`,
			explanation: `pass its .graphql, .graphqls or .gql in the same command`,
		},
		{
			name:        "graph.required_by_federation",
			violation:   reports.Violation{Code: "graph.required_by_federation", Source: "graph/schema.graphqls", Details: map[string]string{}},
			headline:    `the GraphQL schema links Apollo Federation, but no graph is declared`,
			explanation: `declare it in provides.graphql.graph`,
		},
		{
			name: "graph.name_taken",
			violation: reports.Violation{Code: "graph.name_taken", Path: "provides;graphql;graph", Source: "contracts/api.yaml",
				Details: map[string]string{"graph": "petstore_graph"}},
			headline:    `graph name "petstore_graph" is already a participant name at provides graphql`,
			explanation: `pick another graph name`,
		},
		{
			name: "graph.name_syntax",
			violation: reports.Violation{Code: "graph.name_syntax", Path: "provides;graphql;graph", Source: "contracts/api.yaml",
				Details: map[string]string{"graph": "Petstore-Graph", "error": "must be snake_case"}},
			headline:    `invalid graph name "Petstore-Graph" at provides graphql`,
			explanation: `must be snake_case`,
		},
		{
			name: "graph.conflict",
			violation: reports.Violation{Code: "graph.conflict", Path: "provides;graphql;graph", Source: "contracts/orders.yaml",
				Details: map[string]string{"graph": "other_graph", "declaredIn": "contracts/api.yaml", "declared": "petstore_graph"}},
			headline:    `graph "other_graph" conflicts with graph "petstore_graph" declared in contracts/api.yaml at provides graphql`,
			explanation: `a contract belongs to one graph: declare the same name in every file`,
		},
		{
			name: "operation.name_syntax",
			violation: reports.Violation{Code: "operation.name_syntax", Path: "consumes;petstore_api;graphql;query;pets(first: 10)", Source: "contracts/web.yaml",
				Details: map[string]string{"name": "pets(first: 10)"}},
			headline:    `invalid GraphQL field name "pets(first: 10)" at consumes petstore_api graphql query`,
			explanation: `write the field name only; arguments go in "arguments"`,
		},
		{
			name: "schema.key_not_allowed with name on an object schema points to the scalar type",
			violation: reports.Violation{Code: "schema.key_not_allowed", Path: "schemas;Pet;properties;bornAt", Source: "contracts/web.yaml",
				Details: map[string]string{"key": "name", "type": "object"}},
			headline:    `"name" is not allowed on an object schema at schemas Pet properties bornAt`,
			explanation: `use it with "type: scalar" (GraphQL contracts only)`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			headline, explanation := reports.FormatViolationLine(test.violation, modules)

			assert.Equal(t, test.headline, headline)
			assert.Equal(t, test.explanation, explanation)
		})
	}
}

func TestViolationLineLeavesTheCodesOfOtherProtocolsToThem(t *testing.T) {
	_, _, known := graphql.Module().Report.ViolationLine(reports.Violation{Code: "endpoint.syntax", Path: "provides;rest;/users/{userId}"})

	assert.False(t, known)
}
