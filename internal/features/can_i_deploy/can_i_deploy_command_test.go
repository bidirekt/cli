package can_i_deploy

import (
	"bytes"
	"testing"

	"github.com/bidirekt/cli/internal/paint"
	"github.com/stretchr/testify/assert"
)

func TestFormatDeployableLineNamesTheVersionUnderCheck(t *testing.T) {
	line := formatDeployableLine(paint.For(&bytes.Buffer{}), "petstore_web", "3.0.0", "production")

	assert.Equal(t, "petstore_web 3.0.0 can be deployed to production", line)
}

func TestFormatDeployableLinePaintsItGreen(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "1")

	line := formatDeployableLine(paint.For(&bytes.Buffer{}), "petstore_web", "3.0.0", "production")

	assert.Equal(t, "\x1b[32mpetstore_web 3.0.0 can be deployed to production\x1b[0m", line)
}

func TestFormatNotDeployableReportWalksTheTree(t *testing.T) {
	apiVersion := "4.0.0"
	paymentsVersion := "1.4.0"
	results := map[string]CanIDeployResult{
		"petstore_api": {
			Deployable:         false,
			ParticipantVersion: &apiVersion,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/pets": {
					"post": {
						"request": {
							{Reason: "property_optional_in_consumer_required_in_provider", Role: "consumer", Details: map[string]string{
								"property": "$.name", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
							{Reason: "property_missing_in_consumer", Role: "consumer", Details: map[string]string{
								"property": "$.ownerId", "propertyType": "integer", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
				},
				"/pets/*": {
					"get": {
						"200": {
							{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
								"property": "$.photoUrl", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
							{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
								"property": "$.weight", "consumerPropertyType": "integer", "providerPropertyType": "float",
								"consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
				},
			},
		},
		"petstore_reviews": {
			Deployable: false,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/reviews/*": {
					"get": {
						"200": {
							{Reason: "provider_resource_not_found", Role: "consumer"},
						},
					},
				},
			},
		},
		"petstore_payments": {
			Deployable:         true,
			ParticipantVersion: &paymentsVersion,
			Endpoints:          map[string]map[string]map[string][]ContractBreak{},
		},
	}

	report := formatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web", "2.0.0", "production", results)

	assert.Equal(t, `petstore_web 2.0.0 cannot be deployed to production

petstore_api (4.0.0, deployed):
  POST /pets
    request:
      - petstore_web sends "$.name" only sometimes, but petstore_api requires it → always send it
      - petstore_web doesn't send "$.ownerId", but petstore_api requires it → send it
  GET /pets/*
    response 200:
      - petstore_web reads "$.photoUrl", but petstore_api doesn't provide it → stop reading it, or mark it optional
      - petstore_web reads "$.weight" as integer, but petstore_api provides float → read it as float

petstore_reviews:
  GET /reviews/*
    response 200:
      - petstore_web calls GET /reviews/*, but petstore_reviews doesn't provide it → stop calling it, or wait until petstore_reviews publishes it
`, report)
}

func TestFormatNotDeployableReportSpeaksFromTheProviderSide(t *testing.T) {
	webVersion := "1.0.0"
	results := map[string]CanIDeployResult{
		"petstore_web": {
			Deployable:         false,
			ParticipantVersion: &webVersion,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/pets/*": {
					"get": {
						"200": {
							{Reason: "property_optional_in_provider_required_in_consumer", Role: "provider", Details: map[string]string{
								"property": "$.status", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
				},
			},
		},
	}

	report := formatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_api", "3.0.0", "production", results)

	assert.Equal(t, `petstore_api 3.0.0 cannot be deployed to production

petstore_web (1.0.0, deployed):
  GET /pets/*
    response 200:
      - petstore_api provides "$.status" only sometimes, but petstore_web requires it → keep it required
`, report)
}

func TestFormatNotDeployableReportOmitsVersionWhenNull(t *testing.T) {
	results := map[string]CanIDeployResult{
		"petstore_reviews": {
			Deployable: false,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/reviews/*": {
					"get": {
						"200": {
							{Reason: "provider_resource_not_found", Role: "consumer"},
						},
					},
				},
			},
		},
	}

	report := formatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web", "2.0.0", "production", results)

	assert.Equal(t, `petstore_web 2.0.0 cannot be deployed to production

petstore_reviews:
  GET /reviews/*
    response 200:
      - petstore_web calls GET /reviews/*, but petstore_reviews doesn't provide it → stop calling it, or wait until petstore_reviews publishes it
`, report)
}

func TestFormatNotDeployableReportPaintsTheHeadlineRed(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "1")

	results := map[string]CanIDeployResult{
		"petstore_reviews": {
			Deployable: false,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/reviews/*": {
					"get": {
						"200": {
							{Reason: "provider_resource_not_found", Role: "consumer"},
						},
					},
				},
			},
		},
	}

	report := formatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web", "2.0.0", "production", results)

	assert.Equal(t, "\x1b[31mpetstore_web 2.0.0 cannot be deployed to production\x1b[0m\n"+
		"\n"+
		"petstore_reviews:\n"+
		"  GET /reviews/*\n"+
		"    response 200:\n"+
		"      - petstore_web calls GET /reviews/*, but petstore_reviews doesn't provide it → stop calling it, or wait until petstore_reviews publishes it\n",
		report)
}

func TestFormatBreakLine(t *testing.T) {
	webGetsPet := checkedInteraction{participant: "petstore_web", counterpart: "petstore_api", environment: "production", method: "GET", endpoint: "/pets/*"}
	apiServesPet := checkedInteraction{participant: "petstore_api", counterpart: "petstore_web", environment: "production", method: "GET", endpoint: "/pets/*"}
	webPostsPet := checkedInteraction{participant: "petstore_web", counterpart: "petstore_api", environment: "production", method: "POST", endpoint: "/pets", isRequest: true}
	apiReceivesPet := checkedInteraction{participant: "petstore_api", counterpart: "petstore_web", environment: "production", method: "POST", endpoint: "/pets", isRequest: true}
	webGetsReview := checkedInteraction{participant: "petstore_web", counterpart: "petstore_reviews", environment: "production", method: "GET", endpoint: "/reviews/*"}

	tests := []struct {
		name          string
		interaction   checkedInteraction
		contractBreak ContractBreak
		line          string
	}{
		{
			name:        "property_missing_in_provider as consumer on a response",
			interaction: webGetsPet,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
				"property": "$.photoUrl", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: `petstore_web reads "$.photoUrl", but petstore_api doesn't provide it → stop reading it, or mark it optional`,
		},
		{
			name:        "property_missing_in_provider as provider on a response",
			interaction: apiServesPet,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "provider", Details: map[string]string{
				"property": "$.photoUrl", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: `petstore_api doesn't provide "$.photoUrl", but petstore_web reads it → keep providing it`,
		},
		{
			name:        "property_optional_in_provider_required_in_consumer as consumer on a response",
			interaction: webGetsPet,
			contractBreak: ContractBreak{Reason: "property_optional_in_provider_required_in_consumer", Role: "consumer", Details: map[string]string{
				"property": "$.status", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: `petstore_web requires "$.status", but petstore_api only sometimes provides it → mark it optional`,
		},
		{
			name:        "property_optional_in_provider_required_in_consumer as provider on a response",
			interaction: apiServesPet,
			contractBreak: ContractBreak{Reason: "property_optional_in_provider_required_in_consumer", Role: "provider", Details: map[string]string{
				"property": "$.status", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: `petstore_api provides "$.status" only sometimes, but petstore_web requires it → keep it required`,
		},
		{
			name:        "property_missing_in_consumer as consumer on a request",
			interaction: webPostsPet,
			contractBreak: ContractBreak{Reason: "property_missing_in_consumer", Role: "consumer", Details: map[string]string{
				"property": "$.ownerId", "propertyType": "integer", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: `petstore_web doesn't send "$.ownerId", but petstore_api requires it → send it`,
		},
		{
			name:        "property_missing_in_consumer as provider on a request",
			interaction: apiReceivesPet,
			contractBreak: ContractBreak{Reason: "property_missing_in_consumer", Role: "provider", Details: map[string]string{
				"property": "$.ownerId", "propertyType": "integer", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: `petstore_api requires "$.ownerId", but petstore_web doesn't send it → make it optional`,
		},
		{
			name:        "property_optional_in_consumer_required_in_provider as consumer on a request",
			interaction: webPostsPet,
			contractBreak: ContractBreak{Reason: "property_optional_in_consumer_required_in_provider", Role: "consumer", Details: map[string]string{
				"property": "$.name", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: `petstore_web sends "$.name" only sometimes, but petstore_api requires it → always send it`,
		},
		{
			name:        "property_optional_in_consumer_required_in_provider as provider on a request",
			interaction: apiReceivesPet,
			contractBreak: ContractBreak{Reason: "property_optional_in_consumer_required_in_provider", Role: "provider", Details: map[string]string{
				"property": "$.name", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: `petstore_api requires "$.name", but petstore_web sends it only sometimes → make it optional`,
		},
		{
			name:        "property_type_mismatch as consumer on a response",
			interaction: webGetsPet,
			contractBreak: ContractBreak{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
				"property": "$.weight", "consumerPropertyType": "integer", "providerPropertyType": "float",
				"consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: `petstore_web reads "$.weight" as integer, but petstore_api provides float → read it as float`,
		},
		{
			name:        "property_type_mismatch as provider on a response",
			interaction: apiServesPet,
			contractBreak: ContractBreak{Reason: "property_type_mismatch", Role: "provider", Details: map[string]string{
				"property": "$.weight", "consumerPropertyType": "integer", "providerPropertyType": "float",
				"consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: `petstore_api provides "$.weight" as float, but petstore_web reads integer → provide integer`,
		},
		{
			name:        "property_type_mismatch as consumer on a request",
			interaction: webPostsPet,
			contractBreak: ContractBreak{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
				"property": "$.age", "consumerPropertyType": "string", "providerPropertyType": "integer",
				"consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: `petstore_web sends "$.age" as string, but petstore_api expects integer → send integer`,
		},
		{
			name:        "property_type_mismatch as provider on a request",
			interaction: apiReceivesPet,
			contractBreak: ContractBreak{Reason: "property_type_mismatch", Role: "provider", Details: map[string]string{
				"property": "$.age", "consumerPropertyType": "string", "providerPropertyType": "integer",
				"consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: `petstore_api expects "$.age" as integer, but petstore_web sends string → accept string`,
		},
		{
			name:          "provider_resource_not_found as consumer",
			interaction:   webGetsReview,
			contractBreak: ContractBreak{Reason: "provider_resource_not_found", Role: "consumer"},
			line:          `petstore_web calls GET /reviews/*, but petstore_reviews doesn't provide it → stop calling it, or wait until petstore_reviews publishes it`,
		},
		{
			name:        "provider_resource_not_deployed_in_environment as consumer lists where the provider is deployed",
			interaction: webGetsReview,
			contractBreak: ContractBreak{Reason: "provider_resource_not_deployed_in_environment", Role: "consumer", Details: map[string]string{
				"deployedEnvironments": "staging, qa",
			}},
			line: `petstore_web calls GET /reviews/*, but petstore_reviews is not deployed in production (deployed in: staging, qa) → deploy petstore_reviews first`,
		},
		{
			name:          "provider_resource_not_deployed_in_environment as consumer without deployedEnvironments",
			interaction:   webGetsReview,
			contractBreak: ContractBreak{Reason: "provider_resource_not_deployed_in_environment", Role: "consumer"},
			line:          `petstore_web calls GET /reviews/*, but petstore_reviews is not deployed in production → deploy petstore_reviews first`,
		},
		{
			name:          "provider_resource_removed_but_still_consumed as provider",
			interaction:   apiServesPet,
			contractBreak: ContractBreak{Reason: "provider_resource_removed_but_still_consumed", Role: "provider"},
			line:          `petstore_api removed GET /pets/*, but petstore_web still calls it → keep it until petstore_web stops calling it`,
		},
		{
			name:        "missing role falls back to the raw line instead of inferring it from consumerName",
			interaction: webGetsPet,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Details: map[string]string{
				"property": "$.photoUrl", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: `property_missing_in_provider (consumerName: petstore_web, property: $.photoUrl, propertyType: string, providerName: petstore_api)`,
		},
		{
			name:        "unknown role falls back to the raw line",
			interaction: webGetsPet,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "observer", Details: map[string]string{
				"property": "$.photoUrl", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: `property_missing_in_provider (consumerName: petstore_web, property: $.photoUrl, propertyType: string, providerName: petstore_api)`,
		},
		{
			name:          "unknown reason falls back to the bare reason",
			interaction:   webGetsPet,
			contractBreak: ContractBreak{Reason: "some_future_reason", Role: "consumer"},
			line:          `some_future_reason`,
		},
		{
			name:        "property_missing_in_provider on a request is outside the catalog and falls back to the raw line",
			interaction: webPostsPet,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
				"property": "$.name", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: `property_missing_in_provider (consumerName: petstore_web, property: $.name, propertyType: string, providerName: petstore_api)`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.line, formatBreakLine(test.interaction, test.contractBreak))
		})
	}
}
