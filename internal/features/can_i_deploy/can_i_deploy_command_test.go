package can_i_deploy

import (
	"bytes"
	"testing"

	"github.com/bidirekt/cli/internal/paint"
	"github.com/stretchr/testify/assert"
)

func TestFormatNotDeployableReportWalksTheTree(t *testing.T) {
	version := "3.1.0"
	results := map[string]CanIDeployResult{
		"payments-api": {
			Deployable:         false,
			ParticipantVersion: &version,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/payments/*": {
					"get": {
						"request": {
							{Reason: "property_missing_in_provider", Details: map[string]string{"property": "currency"}},
						},
						"200": {
							{Reason: "property_type_mismatch", Details: map[string]string{
								"property":             "amount",
								"consumerPropertyType": "string",
								"providerPropertyType": "number",
							}},
						},
					},
				},
			},
		},
		"users": {
			Deployable:         true,
			ParticipantVersion: &version,
			Endpoints:          map[string]map[string]map[string][]ContractBreak{},
		},
	}

	report := formatNotDeployableReport(paint.For(&bytes.Buffer{}), "payments-web", "production", results)

	assert.Equal(t, `payments-web cannot be deployed to production

payments-api (3.1.0):
  GET /payments/*
    request:
      - property "currency" is missing in provider
    response 200:
      - property "amount" type mismatch — consumer has string, provider has number
`, report)
}

func TestFormatNotDeployableReportOmitsVersionWhenNull(t *testing.T) {
	results := map[string]CanIDeployResult{
		"payments-api": {
			Deployable: false,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/payments/*": {
					"get": {
						"200": {
							{Reason: "provider_resource_not_found"},
						},
					},
				},
			},
		},
	}

	report := formatNotDeployableReport(paint.For(&bytes.Buffer{}), "payments-web", "production", results)

	assert.Equal(t, `payments-web cannot be deployed to production

payments-api:
  GET /payments/*
    response 200:
      - no matching resource in provider
`, report)
}

func TestFormatNotDeployableReportPaintsTheHeadlineRed(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "1")

	results := map[string]CanIDeployResult{
		"payments-api": {
			Deployable: false,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/payments/*": {
					"get": {
						"200": {
							{Reason: "provider_resource_not_found"},
						},
					},
				},
			},
		},
	}

	report := formatNotDeployableReport(paint.For(&bytes.Buffer{}), "payments-web", "production", results)

	assert.Equal(t, "\x1b[31mpayments-web cannot be deployed to production\x1b[0m\n"+
		"\n"+
		"payments-api:\n"+
		"  GET /payments/*\n"+
		"    response 200:\n"+
		"      - no matching resource in provider\n",
		report)
}

func TestFormatBreakLine(t *testing.T) {
	t.Run("provider_resource_not_found", func(t *testing.T) {
		line := formatBreakLine("production", ContractBreak{
			Reason: "provider_resource_not_found",
		})

		assert.Equal(t, `no matching resource in provider`, line)
	})

	t.Run("provider_resource_removed_but_still_consumed", func(t *testing.T) {
		line := formatBreakLine("production", ContractBreak{
			Reason: "provider_resource_removed_but_still_consumed",
		})

		assert.Equal(t, `resource removed but still consumed`, line)
	})

	t.Run("provider_resource_not_deployed_in_environment", func(t *testing.T) {
		line := formatBreakLine("production", ContractBreak{
			Reason:  "provider_resource_not_deployed_in_environment",
			Details: map[string]string{"deployedEnvironments": "staging, qa"},
		})

		assert.Equal(t, `provider is not deployed in "production" (deployed in: staging, qa)`, line)
	})

	t.Run("provider_resource_not_deployed_in_environment without deployedEnvironments", func(t *testing.T) {
		line := formatBreakLine("production", ContractBreak{
			Reason: "provider_resource_not_deployed_in_environment",
		})

		assert.Equal(t, `provider is not deployed in "production"`, line)
	})

	t.Run("property_missing_in_provider", func(t *testing.T) {
		line := formatBreakLine("production", ContractBreak{
			Reason:  "property_missing_in_provider",
			Details: map[string]string{"property": "amount"},
		})

		assert.Equal(t, `property "amount" is missing in provider`, line)
	})

	t.Run("property_missing_in_consumer", func(t *testing.T) {
		line := formatBreakLine("production", ContractBreak{
			Reason:  "property_missing_in_consumer",
			Details: map[string]string{"property": "amount"},
		})

		assert.Equal(t, `property "amount" is missing in consumer`, line)
	})

	t.Run("property_optional_in_provider_required_in_consumer", func(t *testing.T) {
		line := formatBreakLine("production", ContractBreak{
			Reason:  "property_optional_in_provider_required_in_consumer",
			Details: map[string]string{"property": "amount"},
		})

		assert.Equal(t, `property "amount" is optional in provider but required in consumer`, line)
	})

	t.Run("property_optional_in_consumer_required_in_provider", func(t *testing.T) {
		line := formatBreakLine("production", ContractBreak{
			Reason:  "property_optional_in_consumer_required_in_provider",
			Details: map[string]string{"property": "amount"},
		})

		assert.Equal(t, `property "amount" is optional in consumer but required in provider`, line)
	})

	t.Run("property_type_mismatch uses the role-resolved types", func(t *testing.T) {
		line := formatBreakLine("production", ContractBreak{
			Reason: "property_type_mismatch",
			Details: map[string]string{
				"property":             "amount",
				"consumerPropertyType": "string",
				"providerPropertyType": "number",
			},
		})

		assert.Equal(t, `property "amount" type mismatch — consumer has string, provider has number`, line)
	})

	t.Run("unknown reason falls back to the verbatim reason with details", func(t *testing.T) {
		line := formatBreakLine("production", ContractBreak{
			Reason:  "some_future_reason",
			Details: map[string]string{"property": "amount", "extra": "x"},
		})

		assert.Equal(t, `some_future_reason (extra: x, property: amount)`, line)
	})
}
