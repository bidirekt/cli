package reports

import (
	"bytes"
	"testing"

	"github.com/bidirekt/cli/internal/paint"
	"github.com/stretchr/testify/assert"
)

func TestFormatDeployableLineNamesTheCheckedSide(t *testing.T) {
	line := FormatDeployableLine(paint.For(&bytes.Buffer{}), "petstore_web 3.0.0", "production")

	assert.Equal(t, "petstore_web 3.0.0 can be deployed to production", line)
}

func TestFormatDeployableLinePaintsItGreen(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "1")

	line := FormatDeployableLine(paint.For(&bytes.Buffer{}), "petstore_web 3.0.0", "production")

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

	report := FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 2.0.0", "petstore_web", "production", results)

	assert.Equal(t, `petstore_web 2.0.0 cannot be deployed to production

petstore_api (4.0.0, deployed):
  POST /pets
    request:
      - petstore_web sends "$.name" only sometimes, but petstore_api requires it
        → petstore_web: always send it
        → petstore_api: handle its absence in the code, then mark it optional
      - petstore_web doesn't send "$.ownerId", but petstore_api requires it
        → petstore_web: send it
        → petstore_api: stop requiring it, then mark it optional
  GET /pets/*
    response 200:
      - petstore_web reads "$.photoUrl", but petstore_api doesn't provide it
        → petstore_web: stop reading it, or handle its absence in the code and mark it optional
        → petstore_api: provide it
      - petstore_web reads "$.weight" as integer, but petstore_api provides float
        → petstore_web: read it as float
        → petstore_api: provide it as integer

petstore_reviews:
  GET /reviews/*
    response 200:
      - petstore_web calls GET /reviews/*, but petstore_reviews doesn't provide it
        → petstore_web: stop calling it
        → petstore_reviews: publish it
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

	report := FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_api 3.0.0", "petstore_api", "production", results)

	assert.Equal(t, `petstore_api 3.0.0 cannot be deployed to production

petstore_web (1.0.0, deployed):
  GET /pets/*
    response 200:
      - petstore_api provides "$.status" only sometimes, but petstore_web requires it
        → petstore_api: always provide it
        → petstore_web: handle its absence in the code, then mark it optional
`, report)
}

func TestFormatNotDeployableReportPrintsARemovedEndpointOnce(t *testing.T) {
	results := map[string]CanIDeployResult{
		"petstore_web": {
			Deployable: false,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/pets": {
					"post": {
						"request": {{Reason: "provider_resource_removed_but_still_consumed", Role: "provider"}},
						"201":     {{Reason: "provider_resource_removed_but_still_consumed", Role: "provider"}},
						"400":     {{Reason: "provider_resource_removed_but_still_consumed", Role: "provider"}},
					},
				},
			},
		},
	}

	report := FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_api 3.0.0", "petstore_api", "production", results)

	assert.Equal(t, `petstore_api 3.0.0 cannot be deployed to production

petstore_web:
  POST /pets
    - petstore_api removed POST /pets, but petstore_web still calls it
      → petstore_api: keep it until petstore_web stops calling it
      → petstore_web: stop calling it
`, report)
}

func TestFormatNotDeployableReportKeepsOtherBreaksOfARemovedEndpointUnderTheirBody(t *testing.T) {
	results := map[string]CanIDeployResult{
		"petstore_web": {
			Deployable: false,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/pets": {
					"post": {
						"request": {{Reason: "provider_resource_removed_but_still_consumed", Role: "provider"}},
						"201": {
							{Reason: "provider_resource_removed_but_still_consumed", Role: "provider"},
							{Reason: "property_optional_in_provider_required_in_consumer", Role: "provider", Details: map[string]string{
								"property": "$.status", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
				},
			},
		},
	}

	report := FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_api 3.0.0", "petstore_api", "production", results)

	assert.Equal(t, `petstore_api 3.0.0 cannot be deployed to production

petstore_web:
  POST /pets
    - petstore_api removed POST /pets, but petstore_web still calls it
      → petstore_api: keep it until petstore_web stops calling it
      → petstore_web: stop calling it
    response 201:
      - petstore_api provides "$.status" only sometimes, but petstore_web requires it
        → petstore_api: always provide it
        → petstore_web: handle its absence in the code, then mark it optional
`, report)
}

func TestFormatNotDeployableReportPrintsEachRemovedEndpointOnce(t *testing.T) {
	results := map[string]CanIDeployResult{
		"petstore_web": {
			Deployable: false,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/pets": {
					"post": {
						"request": {{Reason: "provider_resource_removed_but_still_consumed", Role: "provider"}},
						"201":     {{Reason: "provider_resource_removed_but_still_consumed", Role: "provider"}},
					},
				},
				"/pets/*": {
					"delete": {
						"204": {{Reason: "provider_resource_removed_but_still_consumed", Role: "provider"}},
					},
				},
			},
		},
	}

	report := FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_api 3.0.0", "petstore_api", "production", results)

	assert.Equal(t, `petstore_api 3.0.0 cannot be deployed to production

petstore_web:
  POST /pets
    - petstore_api removed POST /pets, but petstore_web still calls it
      → petstore_api: keep it until petstore_web stops calling it
      → petstore_web: stop calling it
  DELETE /pets/*
    - petstore_api removed DELETE /pets/*, but petstore_web still calls it
      → petstore_api: keep it until petstore_web stops calling it
      → petstore_web: stop calling it
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

	report := FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 2.0.0", "petstore_web", "production", results)

	assert.Equal(t, `petstore_web 2.0.0 cannot be deployed to production

petstore_reviews:
  GET /reviews/*
    response 200:
      - petstore_web calls GET /reviews/*, but petstore_reviews doesn't provide it
        → petstore_web: stop calling it
        → petstore_reviews: publish it
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

	report := FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 2.0.0", "petstore_web", "production", results)

	assert.Equal(t, "\x1b[31mpetstore_web 2.0.0 cannot be deployed to production\x1b[0m\n"+
		"\n"+
		"petstore_reviews:\n"+
		"  GET /reviews/*\n"+
		"    response 200:\n"+
		"      - petstore_web calls GET /reviews/*, but petstore_reviews doesn't provide it\n"+
		"        → petstore_web: stop calling it\n"+
		"        → petstore_reviews: publish it\n",
		report)
}

func TestFormatBreakLine(t *testing.T) {
	webGetsPet := checkedInteraction{participant: "petstore_web", counterpart: "petstore_api", environment: "production", method: "GET", endpoint: "/pets/*"}
	apiServesPet := checkedInteraction{participant: "petstore_api", counterpart: "petstore_web", environment: "production", method: "GET", endpoint: "/pets/*"}
	webPostsPet := checkedInteraction{participant: "petstore_web", counterpart: "petstore_api", environment: "production", method: "POST", endpoint: "/pets", isRequest: true}
	apiReceivesPet := checkedInteraction{participant: "petstore_api", counterpart: "petstore_web", environment: "production", method: "POST", endpoint: "/pets", isRequest: true}
	webGetsReview := checkedInteraction{participant: "petstore_web", counterpart: "petstore_reviews", environment: "production", method: "GET", endpoint: "/reviews/*"}
	spaGetsFiles := checkedInteraction{participant: "spa", counterpart: "api", environment: "production", method: "GET", endpoint: "/files"}
	spaGetsSession := checkedInteraction{participant: "spa", counterpart: "api", environment: "production", method: "GET", endpoint: "/session"}
	apiServesSession := checkedInteraction{participant: "api", counterpart: "spa", environment: "production", method: "GET", endpoint: "/session"}
	spaPostsSession := checkedInteraction{participant: "spa", counterpart: "api", environment: "production", method: "POST", endpoint: "/session", isRequest: true}
	apiReceivesSession := checkedInteraction{participant: "api", counterpart: "spa", environment: "production", method: "POST", endpoint: "/session", isRequest: true}

	tests := []struct {
		name          string
		interaction   checkedInteraction
		contractBreak ContractBreak
		line          breakLine
	}{
		{
			name:        "property_missing_in_provider as consumer on a response",
			interaction: webGetsPet,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
				"consumerField": "Pet.photoUrl", "property": "$.photoUrl", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{
				text: `petstore_web reads "$.photoUrl" (Pet.photoUrl), but petstore_api doesn't provide it`,
				fixes: []string{
					"petstore_web: stop reading it, or handle its absence in the code and mark it optional",
					"petstore_api: provide it",
				},
			},
		},
		{
			name:        "property_missing_in_provider as provider on a response",
			interaction: apiServesPet,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "provider", Details: map[string]string{
				"consumerField": "Pet.photoUrl", "property": "$.photoUrl", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{
				text: `petstore_api doesn't provide "$.photoUrl", but petstore_web reads it`,
				fixes: []string{
					"petstore_api: provide it",
					"petstore_web: stop reading it, or handle its absence in the code and mark it optional at Pet.photoUrl",
				},
			},
		},
		{
			name:        "property_optional_in_provider_required_in_consumer as consumer on a response",
			interaction: webGetsPet,
			contractBreak: ContractBreak{Reason: "property_optional_in_provider_required_in_consumer", Role: "consumer", Details: map[string]string{
				"consumerField": "Pet.status", "providerField": "PetRecord.status", "property": "$.status", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{
				text: `petstore_web requires "$.status" (Pet.status), but petstore_api only sometimes provides it`,
				fixes: []string{
					"petstore_web: handle its absence in the code, then mark it optional",
					"petstore_api: always provide it at PetRecord.status",
				},
			},
		},
		{
			name:        "property_optional_in_provider_required_in_consumer as provider on a response",
			interaction: apiServesPet,
			contractBreak: ContractBreak{Reason: "property_optional_in_provider_required_in_consumer", Role: "provider", Details: map[string]string{
				"consumerField": "Pet.status", "providerField": "PetRecord.status", "property": "$.status", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{
				text: `petstore_api provides "$.status" (PetRecord.status) only sometimes, but petstore_web requires it`,
				fixes: []string{
					"petstore_api: always provide it",
					"petstore_web: handle its absence in the code, then mark it optional at Pet.status",
				},
			},
		},
		{
			name:        "property_type_mismatch as consumer on a response",
			interaction: webGetsPet,
			contractBreak: ContractBreak{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
				"consumerField": "Pet.weight", "providerField": "PetRecord.weight", "property": "$.weight", "consumerPropertyType": "integer", "providerPropertyType": "float",
				"consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{
				text: `petstore_web reads "$.weight" (Pet.weight) as integer, but petstore_api provides float`,
				fixes: []string{
					"petstore_web: read it as float",
					"petstore_api: provide it as integer at PetRecord.weight",
				},
			},
		},
		{
			name:        "property_type_mismatch as provider on a response",
			interaction: apiServesPet,
			contractBreak: ContractBreak{Reason: "property_type_mismatch", Role: "provider", Details: map[string]string{
				"consumerField": "Pet.weight", "providerField": "PetRecord.weight", "property": "$.weight", "consumerPropertyType": "integer", "providerPropertyType": "float",
				"consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{
				text: `petstore_api provides "$.weight" (PetRecord.weight) as float, but petstore_web reads integer`,
				fixes: []string{
					"petstore_api: provide it as integer",
					"petstore_web: read it as float at Pet.weight",
				},
			},
		},
		{
			name:        "property_nullable_in_provider_not_nullable_in_consumer as consumer on a response",
			interaction: webGetsPet,
			contractBreak: ContractBreak{Reason: "property_nullable_in_provider_not_nullable_in_consumer", Role: "consumer", Details: map[string]string{
				"consumerField": "Pet.tag", "providerField": "PetRecord.tag", "property": "$.tag", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{
				text: `petstore_web doesn't accept null at "$.tag" (Pet.tag), but petstore_api may provide null`,
				fixes: []string{
					"petstore_web: handle null in the code, then mark it nullable",
					"petstore_api: stop providing null at PetRecord.tag",
				},
			},
		},
		{
			name:        "property_nullable_in_provider_not_nullable_in_consumer as provider on a response",
			interaction: apiServesPet,
			contractBreak: ContractBreak{Reason: "property_nullable_in_provider_not_nullable_in_consumer", Role: "provider", Details: map[string]string{
				"consumerField": "Pet.tag", "providerField": "PetRecord.tag", "property": "$.tag", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{
				text: `petstore_api may provide null at "$.tag" (PetRecord.tag), but petstore_web doesn't accept it`,
				fixes: []string{
					"petstore_api: stop providing null",
					"petstore_web: handle null in the code, then mark it nullable at Pet.tag",
				},
			},
		},
		{
			name:        "property_missing_in_consumer as consumer on a request",
			interaction: webPostsPet,
			contractBreak: ContractBreak{Reason: "property_missing_in_consumer", Role: "consumer", Details: map[string]string{
				"providerField": "NewPetRecord.ownerId", "property": "$.ownerId", "propertyType": "integer", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{
				text: `petstore_web doesn't send "$.ownerId", but petstore_api requires it`,
				fixes: []string{
					"petstore_web: send it",
					"petstore_api: stop requiring it, then mark it optional at NewPetRecord.ownerId",
				},
			},
		},
		{
			name:        "property_missing_in_consumer as provider on a request",
			interaction: apiReceivesPet,
			contractBreak: ContractBreak{Reason: "property_missing_in_consumer", Role: "provider", Details: map[string]string{
				"providerField": "NewPetRecord.ownerId", "property": "$.ownerId", "propertyType": "integer", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{
				text: `petstore_api requires "$.ownerId" (NewPetRecord.ownerId), but petstore_web doesn't send it`,
				fixes: []string{
					"petstore_api: stop requiring it, then mark it optional",
					"petstore_web: send it",
				},
			},
		},
		{
			name:        "property_optional_in_consumer_required_in_provider as consumer on a request",
			interaction: webPostsPet,
			contractBreak: ContractBreak{Reason: "property_optional_in_consumer_required_in_provider", Role: "consumer", Details: map[string]string{
				"consumerField": "NewPet.name", "providerField": "NewPetRecord.name", "property": "$.name", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{
				text: `petstore_web sends "$.name" (NewPet.name) only sometimes, but petstore_api requires it`,
				fixes: []string{
					"petstore_web: always send it",
					"petstore_api: handle its absence in the code, then mark it optional at NewPetRecord.name",
				},
			},
		},
		{
			name:        "property_optional_in_consumer_required_in_provider as provider on a request",
			interaction: apiReceivesPet,
			contractBreak: ContractBreak{Reason: "property_optional_in_consumer_required_in_provider", Role: "provider", Details: map[string]string{
				"consumerField": "NewPet.name", "providerField": "NewPetRecord.name", "property": "$.name", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{
				text: `petstore_api requires "$.name" (NewPetRecord.name), but petstore_web sends it only sometimes`,
				fixes: []string{
					"petstore_api: handle its absence in the code, then mark it optional",
					"petstore_web: always send it at NewPet.name",
				},
			},
		},
		{
			name:        "property_type_mismatch as consumer on a request",
			interaction: webPostsPet,
			contractBreak: ContractBreak{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
				"consumerField": "NewPet.age", "providerField": "NewPetRecord.age", "property": "$.age", "consumerPropertyType": "string", "providerPropertyType": "integer",
				"consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{
				text: `petstore_web sends "$.age" (NewPet.age) as string, but petstore_api expects integer`,
				fixes: []string{
					"petstore_web: send integer",
					"petstore_api: accept string at NewPetRecord.age",
				},
			},
		},
		{
			name:        "property_type_mismatch as provider on a request",
			interaction: apiReceivesPet,
			contractBreak: ContractBreak{Reason: "property_type_mismatch", Role: "provider", Details: map[string]string{
				"consumerField": "NewPet.age", "providerField": "NewPetRecord.age", "property": "$.age", "consumerPropertyType": "string", "providerPropertyType": "integer",
				"consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{
				text: `petstore_api expects "$.age" (NewPetRecord.age) as integer, but petstore_web sends string`,
				fixes: []string{
					"petstore_api: accept string",
					"petstore_web: send integer at NewPet.age",
				},
			},
		},
		{
			name:        "property_nullable_in_consumer_not_nullable_in_provider as consumer on a request",
			interaction: webPostsPet,
			contractBreak: ContractBreak{Reason: "property_nullable_in_consumer_not_nullable_in_provider", Role: "consumer", Details: map[string]string{
				"consumerField": "NewPet.birthDate", "providerField": "NewPetRecord.birthDate", "property": "$.birthDate", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{
				text: `petstore_web may send null at "$.birthDate" (NewPet.birthDate), but petstore_api doesn't accept it`,
				fixes: []string{
					"petstore_web: stop sending null",
					"petstore_api: handle null in the code, then mark it nullable at NewPetRecord.birthDate",
				},
			},
		},
		{
			name:        "property_nullable_in_consumer_not_nullable_in_provider as provider on a request",
			interaction: apiReceivesPet,
			contractBreak: ContractBreak{Reason: "property_nullable_in_consumer_not_nullable_in_provider", Role: "provider", Details: map[string]string{
				"consumerField": "NewPet.birthDate", "providerField": "NewPetRecord.birthDate", "property": "$.birthDate", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{
				text: `petstore_api doesn't accept null at "$.birthDate" (NewPetRecord.birthDate), but petstore_web may send it`,
				fixes: []string{
					"petstore_api: handle null in the code, then mark it nullable",
					"petstore_web: stop sending null at NewPet.birthDate",
				},
			},
		},
		{
			name:        "property_missing_in_provider at the root path names the body",
			interaction: spaGetsFiles,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
				"consumerField": "Order", "property": "$", "propertyType": "object", "consumerName": "spa", "providerName": "api",
			}},
			line: breakLine{
				text: `spa reads the body (Order), but api doesn't provide it`,
				fixes: []string{
					"spa: stop reading it, or handle its absence in the code and mark it optional",
					"api: provide it",
				},
			},
		},
		{
			name:        "property_type_mismatch at the root path names the body",
			interaction: spaGetsFiles,
			contractBreak: ContractBreak{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
				"consumerField": "Order", "providerField": "Orders", "property": "$", "consumerPropertyType": "object", "providerPropertyType": "array<object>",
				"consumerName": "spa", "providerName": "api",
			}},
			line: breakLine{
				text: `spa reads the body (Order) as object, but api provides array<object>`,
				fixes: []string{
					"spa: read it as array<object>",
					"api: provide it as object at Orders",
				},
			},
		},
		{
			name:        "property_missing_in_provider at a map's values reads the parent the way the provider provides it",
			interaction: spaGetsSession,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
				"consumerField": "Session.onboarding{}", "property": "$.onboarding{}", "propertyType": "boolean", "consumerName": "spa", "providerName": "api",
			}},
			line: breakLine{
				text: `spa reads "$.onboarding{}" (Session.onboarding{}), but api doesn't provide it`,
				fixes: []string{
					`spa: read "$.onboarding" the way api provides it`,
					"api: provide it",
				},
			},
		},
		{
			name:        "property_missing_in_provider at a map's values as provider leaves the field off the consumer's parent fix",
			interaction: apiServesSession,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "provider", Details: map[string]string{
				"consumerField": "Session.onboarding{}", "property": "$.onboarding{}", "propertyType": "boolean", "consumerName": "spa", "providerName": "api",
			}},
			line: breakLine{
				text: `api doesn't provide "$.onboarding{}", but spa reads it`,
				fixes: []string{
					"api: provide it",
					`spa: read "$.onboarding" the way api provides it`,
				},
			},
		},
		{
			name:        "property_optional_in_provider_required_in_consumer at an array's items reads the parent the way the provider provides it",
			interaction: spaGetsSession,
			contractBreak: ContractBreak{Reason: "property_optional_in_provider_required_in_consumer", Role: "consumer", Details: map[string]string{
				"consumerField": "Session.tags[]", "providerField": "SessionRecord.tags[]", "property": "$.tags[]", "propertyType": "string", "consumerName": "spa", "providerName": "api",
			}},
			line: breakLine{
				text: `spa requires "$.tags[]" (Session.tags[]), but api only sometimes provides it`,
				fixes: []string{
					`spa: read "$.tags" the way api provides it`,
					"api: always provide it at SessionRecord.tags[]",
				},
			},
		},
		{
			name:        "property_missing_in_provider at the root's items reads the body the way the provider provides it",
			interaction: spaGetsFiles,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
				"consumerField": "Files[]", "property": "$[]", "propertyType": "string", "consumerName": "spa", "providerName": "api",
			}},
			line: breakLine{
				text: `spa reads "$[]" (Files[]), but api doesn't provide it`,
				fixes: []string{
					"spa: read the body the way api provides it",
					"api: provide it",
				},
			},
		},
		{
			name:        "property_missing_in_consumer at a map's values reads the parent the way the consumer sends it",
			interaction: apiReceivesSession,
			contractBreak: ContractBreak{Reason: "property_missing_in_consumer", Role: "provider", Details: map[string]string{
				"providerField": "NewSessionRecord.flags{}", "property": "$.flags{}", "propertyType": "boolean", "consumerName": "spa", "providerName": "api",
			}},
			line: breakLine{
				text: `api requires "$.flags{}" (NewSessionRecord.flags{}), but spa doesn't send it`,
				fixes: []string{
					`api: read "$.flags" the way spa sends it`,
					"spa: send it",
				},
			},
		},
		{
			name:        "property_optional_in_consumer_required_in_provider at a map's values as consumer leaves the field off the provider's parent fix",
			interaction: spaPostsSession,
			contractBreak: ContractBreak{Reason: "property_optional_in_consumer_required_in_provider", Role: "consumer", Details: map[string]string{
				"consumerField": "NewSession.flags{}", "providerField": "NewSessionRecord.flags{}", "property": "$.flags{}", "propertyType": "boolean", "consumerName": "spa", "providerName": "api",
			}},
			line: breakLine{
				text: `spa sends "$.flags{}" (NewSession.flags{}) only sometimes, but api requires it`,
				fixes: []string{
					"spa: always send it",
					`api: read "$.flags" the way spa sends it`,
				},
			},
		},
		{
			name:        "a fix that doesn't mark it optional keeps its words and the other side's field at an array's items",
			interaction: spaGetsSession,
			contractBreak: ContractBreak{Reason: "property_nullable_in_provider_not_nullable_in_consumer", Role: "consumer", Details: map[string]string{
				"consumerField": "Session.tags[]", "providerField": "SessionRecord.tags[]", "property": "$.tags[]", "propertyType": "string", "consumerName": "spa", "providerName": "api",
			}},
			line: breakLine{
				text: `spa doesn't accept null at "$.tags[]" (Session.tags[]), but api may provide null`,
				fixes: []string{
					"spa: handle null in the code, then mark it nullable",
					"api: stop providing null at SessionRecord.tags[]",
				},
			},
		},
		{
			name:          "provider_resource_not_found as consumer",
			interaction:   webGetsReview,
			contractBreak: ContractBreak{Reason: "provider_resource_not_found", Role: "consumer"},
			line: breakLine{
				text: "petstore_web calls GET /reviews/*, but petstore_reviews doesn't provide it",
				fixes: []string{
					"petstore_web: stop calling it",
					"petstore_reviews: publish it",
				},
			},
		},
		{
			name:        "provider_resource_not_deployed_in_environment as consumer lists where the provider is deployed",
			interaction: webGetsReview,
			contractBreak: ContractBreak{Reason: "provider_resource_not_deployed_in_environment", Role: "consumer", Details: map[string]string{
				"deployedEnvironments": "staging, qa",
			}},
			line: breakLine{
				text:  "petstore_web calls GET /reviews/*, but petstore_reviews is not deployed in production (deployed in: staging, qa)",
				fixes: []string{"petstore_reviews: deploy it to production first"},
			},
		},
		{
			name:          "provider_resource_not_deployed_in_environment as consumer without deployedEnvironments",
			interaction:   webGetsReview,
			contractBreak: ContractBreak{Reason: "provider_resource_not_deployed_in_environment", Role: "consumer"},
			line: breakLine{
				text:  "petstore_web calls GET /reviews/*, but petstore_reviews is not deployed in production",
				fixes: []string{"petstore_reviews: deploy it to production first"},
			},
		},
		{
			name:          "provider_resource_removed_but_still_consumed as provider",
			interaction:   apiServesPet,
			contractBreak: ContractBreak{Reason: "provider_resource_removed_but_still_consumed", Role: "provider"},
			line: breakLine{
				text: "petstore_api removed GET /pets/*, but petstore_web still calls it",
				fixes: []string{
					"petstore_api: keep it until petstore_web stops calling it",
					"petstore_web: stop calling it",
				},
			},
		},
		{
			name:        "missing role falls back to the raw line instead of inferring it from consumerName",
			interaction: webGetsPet,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Details: map[string]string{
				"property": "$.photoUrl", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{text: `property_missing_in_provider (consumerName: petstore_web, property: $.photoUrl, propertyType: string, providerName: petstore_api)`},
		},
		{
			name:        "unknown role falls back to the raw line",
			interaction: webGetsPet,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "observer", Details: map[string]string{
				"property": "$.photoUrl", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{text: `property_missing_in_provider (consumerName: petstore_web, property: $.photoUrl, propertyType: string, providerName: petstore_api)`},
		},
		{
			name:        "fallback keeps the raw root path",
			interaction: spaGetsFiles,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "observer", Details: map[string]string{
				"property": "$", "propertyType": "object", "consumerName": "spa", "providerName": "api",
			}},
			line: breakLine{text: `property_missing_in_provider (consumerName: spa, property: $, propertyType: object, providerName: api)`},
		},
		{
			name:        "fallback keeps the raw fields",
			interaction: webGetsPet,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "observer", Details: map[string]string{
				"property": "$.photoUrl", "consumerField": "Pet.photoUrl", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{text: `property_missing_in_provider (consumerField: Pet.photoUrl, consumerName: petstore_web, property: $.photoUrl, providerName: petstore_api)`},
		},
		{
			name:        "checked side without a field keeps the bare path while the other side's fix names its field",
			interaction: webGetsPet,
			contractBreak: ContractBreak{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
				"property": "$.weight", "providerField": "PetRecord.weight", "consumerPropertyType": "integer", "providerPropertyType": "float",
				"consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{
				text: `petstore_web reads "$.weight" as integer, but petstore_api provides float`,
				fixes: []string{
					"petstore_web: read it as float",
					"petstore_api: provide it as integer at PetRecord.weight",
				},
			},
		},
		{
			name:        "other side without a field ends its fix without one",
			interaction: webGetsPet,
			contractBreak: ContractBreak{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
				"property": "$.weight", "consumerField": "Pet.weight", "consumerPropertyType": "integer", "providerPropertyType": "float",
				"consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{
				text: `petstore_web reads "$.weight" (Pet.weight) as integer, but petstore_api provides float`,
				fixes: []string{
					"petstore_web: read it as float",
					"petstore_api: provide it as integer",
				},
			},
		},
		{
			name:          "unknown reason falls back to the bare reason",
			interaction:   webGetsPet,
			contractBreak: ContractBreak{Reason: "some_future_reason", Role: "consumer"},
			line:          breakLine{text: `some_future_reason`},
		},
		{
			name:        "property_missing_in_provider on a request is outside the catalog and falls back to the raw line",
			interaction: webPostsPet,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
				"property": "$.name", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			line: breakLine{text: `property_missing_in_provider (consumerName: petstore_web, property: $.name, propertyType: string, providerName: petstore_api)`},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.line, formatBreakLine(test.interaction, test.contractBreak))
		})
	}
}
