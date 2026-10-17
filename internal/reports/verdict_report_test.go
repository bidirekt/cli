package reports

import (
	"bytes"
	"testing"

	"github.com/bidirekt/cli/internal/paint"
	"github.com/stretchr/testify/assert"
)

func TestFormatDeployableReportNamesTheCounterpartsComparedWithInAlphabeticalOrder(t *testing.T) {
	apiVersion := "4.0.0"
	results := map[string]CanIDeployResult{
		"petstore_reviews": {Deployable: true, Endpoints: map[string]map[string]map[string][]ContractBreak{}},
		"petstore_api":     {Deployable: true, ParticipantVersion: &apiVersion, Endpoints: map[string]map[string]map[string][]ContractBreak{}},
		"petstore_admin":   {Deployable: true, Endpoints: map[string]map[string]map[string][]ContractBreak{}},
	}

	report := FormatDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 3.0.0", "production", results)

	assert.Equal(t, `petstore_web 3.0.0 can be deployed to production
compared with: petstore_admin, petstore_api, petstore_reviews
`, report)
}

func TestFormatDeployableReportSaysWhenNothingIsDeployedToCompareWith(t *testing.T) {
	report := FormatDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 3.0.0", "production", map[string]CanIDeployResult{})

	assert.Equal(t, `petstore_web 3.0.0 can be deployed to production
nothing deployed in production to compare with
`, report)
}

func TestFormatDeployableReportPaintsOnlyTheVerdictGreen(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "1")
	results := map[string]CanIDeployResult{
		"petstore_api": {Deployable: true, Endpoints: map[string]map[string]map[string][]ContractBreak{}},
	}

	report := FormatDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 3.0.0", "production", results)

	assert.Equal(t, "\x1b[32mpetstore_web 3.0.0 can be deployed to production\x1b[0m\ncompared with: petstore_api\n", report)
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
								"consumerField": "NewPet.name", "providerField": "NewPetRecord.name", "property": "$.name", "propertyType": "string",
								"consumerName": "petstore_web", "providerName": "petstore_api",
							}},
							{Reason: "property_missing_in_consumer", Role: "consumer", Details: map[string]string{
								"providerField": "NewPetRecord.ownerId", "property": "$.ownerId", "propertyType": "integer",
								"consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
				},
				"/pets/*": {
					"get": {
						"200": {
							{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.photoUrl", "property": "$.photoUrl", "propertyType": "string",
								"consumerName": "petstore_web", "providerName": "petstore_api",
							}},
							{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.weight", "providerField": "PetRecord.weight", "property": "$.weight",
								"consumerPropertyType": "integer", "providerPropertyType": "float", "consumerName": "petstore_web", "providerName": "petstore_api",
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
  NewPet.name — petstore_web sends it only sometimes, but petstore_api requires it
    → petstore_web: always send it
    → petstore_api: handle its absence in the code, then mark it optional there
      · POST /pets request at "$.name" (petstore_api: NewPetRecord.name)
  NewPetRecord.ownerId — petstore_web doesn't send it, but petstore_api requires it
    → petstore_web: send it
    → petstore_api: stop requiring it, then mark it optional there
      · POST /pets request at "$.ownerId" (petstore_api: NewPetRecord.ownerId)
  Pet.photoUrl — petstore_web reads it, but petstore_api doesn't provide it
    → petstore_web: stop reading it, or handle its absence in the code and mark it optional
    → petstore_api: provide it
      · GET /pets/* response 200 at "$.photoUrl"
  Pet.weight — petstore_web reads it as integer, but petstore_api provides float
    → petstore_web: read it as float
    → petstore_api: provide it as integer there
      · GET /pets/* response 200 at "$.weight" (petstore_api: PetRecord.weight)

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
								"consumerField": "Pet.status", "providerField": "PetRecord.status", "property": "$.status", "propertyType": "string",
								"consumerName": "petstore_web", "providerName": "petstore_api",
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
  PetRecord.status — petstore_api provides it only sometimes, but petstore_web requires it
    → petstore_api: always provide it
    → petstore_web: handle its absence in the code, then mark it optional there
      · GET /pets/* response 200 at "$.status" (petstore_web: Pet.status)
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

func TestFormatNotDeployableReportKeepsTheOtherBreaksOfARemovedEndpoint(t *testing.T) {
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
								"consumerField": "Pet.status", "providerField": "PetRecord.status", "property": "$.status", "propertyType": "string",
								"consumerName": "petstore_web", "providerName": "petstore_api",
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
  PetRecord.status — petstore_api provides it only sometimes, but petstore_web requires it
    → petstore_api: always provide it
    → petstore_web: handle its absence in the code, then mark it optional there
      · POST /pets response 201 at "$.status" (petstore_web: Pet.status)
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

func TestFormatNotDeployableReportSaysWhatTheProviderLacksOnAMethodItHas(t *testing.T) {
	results := map[string]CanIDeployResult{
		"api": {
			Deployable: false,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/api/newsletter/unsubscribe": {
					"post": {
						"200": {
							{Reason: "provider_resource_interaction_not_found", Role: "consumer", Details: map[string]string{"providerStatuses": "204"}},
						},
					},
					"delete": {
						"204": {
							{Reason: "provider_resource_not_found", Role: "consumer"},
						},
					},
				},
				"/api/users/*/restore": {
					"post": {
						"request": {
							{Reason: "provider_resource_interaction_not_found", Role: "consumer", Details: map[string]string{"providerStatuses": "204, 404"}},
						},
					},
				},
			},
		},
	}

	report := FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "site 1.0.0", "site", "production", results)

	assert.Equal(t, `site 1.0.0 cannot be deployed to production

api:
  DELETE /api/newsletter/unsubscribe
    response 204:
      - site calls DELETE /api/newsletter/unsubscribe, but api doesn't provide it
        → site: stop calling it
        → api: publish it
  POST /api/newsletter/unsubscribe
    response 200:
      - site reads response 200, but api answers only 204
        → site: read response 204
        → api: answer 200
  POST /api/users/*/restore
    request:
      - site sends a request body, but api declares none
        → site: stop sending it
        → api: accept a request body
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

func TestFormatNotDeployableReportGathersAFieldsPlacesInOneBlock(t *testing.T) {
	results := map[string]CanIDeployResult{
		"petstore_api": {
			Deployable: false,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/pets/*": {
					"put": {
						"200": {
							{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.weight", "providerField": "PetRecord.weight", "property": "$.weight",
								"consumerPropertyType": "integer", "providerPropertyType": "float", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
					"get": {
						"200": {
							{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.weight", "providerField": "PetRecord.weight", "property": "$.weight",
								"consumerPropertyType": "integer", "providerPropertyType": "float", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
							{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.weight", "providerField": "PetRecord.weight", "property": "$.mother.weight",
								"consumerPropertyType": "integer", "providerPropertyType": "float", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
				},
				"/pets": {
					"post": {
						"201": {
							{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.weight", "providerField": "PetRecord.weight", "property": "$.weight",
								"consumerPropertyType": "integer", "providerPropertyType": "float", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
						"200": {
							{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.weight", "providerField": "PetRecord.weight", "property": "$.weight",
								"consumerPropertyType": "integer", "providerPropertyType": "float", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
					"get": {
						"200": {
							{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.weight", "providerField": "PetSummary.weight", "property": "$[].weight",
								"consumerPropertyType": "integer", "providerPropertyType": "float", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
				},
			},
		},
	}

	report := FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 2.0.0", "petstore_web", "production", results)

	assert.Equal(t, `petstore_web 2.0.0 cannot be deployed to production

petstore_api:
  Pet.weight — petstore_web reads it as integer, but petstore_api provides float (6 places)
    → petstore_web: read it as float
    → petstore_api: provide it as integer there
      · GET /pets response 200 at "$[].weight" (petstore_api: PetSummary.weight)
      · POST /pets response 200 at "$.weight" (petstore_api: PetRecord.weight)
      · POST /pets response 201 at "$.weight" (petstore_api: PetRecord.weight)
      · GET /pets/* response 200 at "$.mother.weight" (petstore_api: PetRecord.weight)
      · GET /pets/* response 200 at "$.weight" (petstore_api: PetRecord.weight)
      · PUT /pets/* response 200 at "$.weight" (petstore_api: PetRecord.weight)
`, report)
}

func TestFormatNotDeployableReportPutsAFieldsReasonsInSeparateBlocks(t *testing.T) {
	results := map[string]CanIDeployResult{
		"petstore_api": {
			Deployable: false,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/pets/*": {
					"get": {
						"200": {
							{Reason: "property_optional_in_provider_required_in_consumer", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.tag", "providerField": "PetRecord.tag", "property": "$.tag", "propertyType": "string",
								"consumerName": "petstore_web", "providerName": "petstore_api",
							}},
							{Reason: "property_nullable_in_provider_not_nullable_in_consumer", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.tag", "providerField": "PetRecord.tag", "property": "$.tag", "propertyType": "string",
								"consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
				},
			},
		},
	}

	report := FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 2.0.0", "petstore_web", "production", results)

	assert.Equal(t, `petstore_web 2.0.0 cannot be deployed to production

petstore_api:
  Pet.tag — petstore_web doesn't accept null, but petstore_api may provide null
    → petstore_web: handle null in the code, then mark it nullable
    → petstore_api: stop providing null there
      · GET /pets/* response 200 at "$.tag" (petstore_api: PetRecord.tag)
  Pet.tag — petstore_web requires it, but petstore_api only sometimes provides it
    → petstore_web: handle its absence in the code, then mark it optional
    → petstore_api: always provide it there
      · GET /pets/* response 200 at "$.tag" (petstore_api: PetRecord.tag)
`, report)
}

func TestFormatNotDeployableReportOrdersBlocksByFieldThenReasonThenTypes(t *testing.T) {
	results := map[string]CanIDeployResult{
		"petstore_api": {
			Deployable: false,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/pets/*": {
					"get": {
						"200": {
							{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.weight", "providerField": "PetRecord.weight", "property": "$.weight",
								"consumerPropertyType": "integer", "providerPropertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
							{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
								"consumerField": "Owner.name", "property": "$.owner.name", "propertyType": "string",
								"consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
				},
				"/pets": {
					"get": {
						"200": {
							{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.weight", "providerField": "PetSummary.weight", "property": "$[].weight",
								"consumerPropertyType": "integer", "providerPropertyType": "float", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
				},
			},
		},
	}

	report := FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 2.0.0", "petstore_web", "production", results)

	assert.Equal(t, `petstore_web 2.0.0 cannot be deployed to production

petstore_api:
  Owner.name — petstore_web reads it, but petstore_api doesn't provide it
    → petstore_web: stop reading it, or handle its absence in the code and mark it optional
    → petstore_api: provide it
      · GET /pets/* response 200 at "$.owner.name"
  Pet.weight — petstore_web reads it as integer, but petstore_api provides float
    → petstore_web: read it as float
    → petstore_api: provide it as integer there
      · GET /pets response 200 at "$[].weight" (petstore_api: PetSummary.weight)
  Pet.weight — petstore_web reads it as integer, but petstore_api provides string
    → petstore_web: read it as string
    → petstore_api: provide it as integer there
      · GET /pets/* response 200 at "$.weight" (petstore_api: PetRecord.weight)
`, report)
}

func TestFormatNotDeployableReportSplitsAFieldsBlocksByDirectionThenRole(t *testing.T) {
	results := map[string]CanIDeployResult{
		"petstore_api": {
			Deployable: false,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/web/pets/*": {
					"get": {
						"200": {
							{Reason: "property_type_mismatch", Role: "provider", Details: map[string]string{
								"consumerField": "PetView.weight", "providerField": "Pet.weight", "property": "$.weight",
								"consumerPropertyType": "integer", "providerPropertyType": "float", "consumerName": "petstore_api", "providerName": "petstore_web",
							}},
						},
					},
				},
				"/pets/*": {
					"get": {
						"200": {
							{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.weight", "providerField": "PetRecord.weight", "property": "$.weight",
								"consumerPropertyType": "integer", "providerPropertyType": "float", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
				},
				"/pets": {
					"post": {
						"request": {
							{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.weight", "providerField": "PetRecord.weight", "property": "$.weight",
								"consumerPropertyType": "integer", "providerPropertyType": "float", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
				},
			},
		},
	}

	report := FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 2.0.0", "petstore_web", "production", results)

	assert.Equal(t, `petstore_web 2.0.0 cannot be deployed to production

petstore_api:
  Pet.weight — petstore_web sends it as integer, but petstore_api expects float
    → petstore_web: send float
    → petstore_api: accept integer there
      · POST /pets request at "$.weight" (petstore_api: PetRecord.weight)
  Pet.weight — petstore_web reads it as integer, but petstore_api provides float
    → petstore_web: read it as float
    → petstore_api: provide it as integer there
      · GET /pets/* response 200 at "$.weight" (petstore_api: PetRecord.weight)
  Pet.weight — petstore_web provides it as float, but petstore_api reads integer
    → petstore_web: provide it as integer
    → petstore_api: read it as float there
      · GET /web/pets/* response 200 at "$.weight" (petstore_api: PetView.weight)
`, report)
}

func TestFormatNotDeployableReportSplitsAFieldsBlocksByFix(t *testing.T) {
	tests := []struct {
		name                     string
		participant, counterpart string
		endpoints                map[string]map[string]map[string][]ContractBreak
		blocks                   string
	}{
		{
			name:        "an array's items the checked provider provides only sometimes to readers of two schemas",
			participant: "api", counterpart: "spa",
			endpoints: map[string]map[string]map[string][]ContractBreak{
				"/a": {"get": {"200": {
					{Reason: "property_optional_in_provider_required_in_consumer", Role: "provider", Details: map[string]string{
						"consumerField": "A.tags[]", "providerField": "SessionRecord.tags[]", "property": "$.tags[]", "propertyType": "string", "consumerName": "spa", "providerName": "api",
					}},
				}}},
				"/b": {"get": {"200": {
					{Reason: "property_optional_in_provider_required_in_consumer", Role: "provider", Details: map[string]string{
						"consumerField": "B.tags[]", "providerField": "SessionRecord.tags[]", "property": "$.tags[]", "propertyType": "string", "consumerName": "spa", "providerName": "api",
					}},
				}}},
			},
			blocks: `  SessionRecord.tags[] — api provides it only sometimes, but spa requires it
    → api: always provide it
    → spa: read A.tags the way api provides it
      · GET /a response 200 at "$.tags[]" (spa: A.tags[])
  SessionRecord.tags[] — api provides it only sometimes, but spa requires it
    → api: always provide it
    → spa: read B.tags the way api provides it
      · GET /b response 200 at "$.tags[]" (spa: B.tags[])
`,
		},
		{
			name:        "a map's values the checked consumer sends only sometimes to readers of two schemas",
			participant: "spa", counterpart: "api",
			endpoints: map[string]map[string]map[string][]ContractBreak{
				"/a": {"post": {"request": {
					{Reason: "property_optional_in_consumer_required_in_provider", Role: "consumer", Details: map[string]string{
						"consumerField": "NewSession.flags{}", "providerField": "ARecord.flags{}", "property": "$.flags{}", "propertyType": "boolean", "consumerName": "spa", "providerName": "api",
					}},
				}}},
				"/b": {"post": {"request": {
					{Reason: "property_optional_in_consumer_required_in_provider", Role: "consumer", Details: map[string]string{
						"consumerField": "NewSession.flags{}", "providerField": "BRecord.flags{}", "property": "$.flags{}", "propertyType": "boolean", "consumerName": "spa", "providerName": "api",
					}},
				}}},
			},
			blocks: `  NewSession.flags{} — spa sends it only sometimes, but api requires it
    → spa: always send it
    → api: read ARecord.flags the way spa sends it
      · POST /a request at "$.flags{}" (api: ARecord.flags{})
  NewSession.flags{} — spa sends it only sometimes, but api requires it
    → spa: always send it
    → api: read BRecord.flags the way spa sends it
      · POST /b request at "$.flags{}" (api: BRecord.flags{})
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results := map[string]CanIDeployResult{
				test.counterpart: {Deployable: false, Endpoints: test.endpoints},
			}

			report := FormatNotDeployableReport(paint.For(&bytes.Buffer{}), test.participant+" 2.0.0", test.participant, "production", results)

			assert.Equal(t, test.participant+" 2.0.0 cannot be deployed to production\n\n"+test.counterpart+":\n"+test.blocks, report)
		})
	}
}

func TestFormatNotDeployableReportPrintsARepeatedPlaceOnce(t *testing.T) {
	results := map[string]CanIDeployResult{
		"petstore_api": {
			Deployable: false,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/pets/*": {
					"get": {
						"200": {
							{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.weight", "providerField": "PetRecord.weight", "property": "$.weight",
								"consumerPropertyType": "integer", "providerPropertyType": "float", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
							{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.weight", "providerField": "LegacyPetRecord.weight", "property": "$.weight",
								"consumerPropertyType": "integer", "providerPropertyType": "float", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
				},
				"/pets": {
					"get": {
						"200": {
							{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.weight", "providerField": "PetRecord.weight", "property": "$[].weight",
								"consumerPropertyType": "integer", "providerPropertyType": "float", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
				},
			},
		},
	}

	report := FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 2.0.0", "petstore_web", "production", results)

	assert.Equal(t, `petstore_web 2.0.0 cannot be deployed to production

petstore_api:
  Pet.weight — petstore_web reads it as integer, but petstore_api provides float (2 places)
    → petstore_web: read it as float
    → petstore_api: provide it as integer there
      · GET /pets response 200 at "$[].weight" (petstore_api: PetRecord.weight)
      · GET /pets/* response 200 at "$.weight" (petstore_api: LegacyPetRecord.weight)
`, report)
}

func TestFormatNotDeployableReportSaysThereWhenAnyPlaceNamesTheOtherSidesField(t *testing.T) {
	results := map[string]CanIDeployResult{
		"petstore_api": {
			Deployable: false,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/pets/*": {
					"get": {
						"200": {
							{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.weight", "providerField": "PetRecord.weight", "property": "$.weight",
								"consumerPropertyType": "integer", "providerPropertyType": "float", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
				},
				"/pets": {
					"get": {
						"200": {
							{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.weight", "property": "$[].weight",
								"consumerPropertyType": "integer", "providerPropertyType": "float", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
				},
			},
		},
	}

	report := FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 2.0.0", "petstore_web", "production", results)

	assert.Equal(t, `petstore_web 2.0.0 cannot be deployed to production

petstore_api:
  Pet.weight — petstore_web reads it as integer, but petstore_api provides float (2 places)
    → petstore_web: read it as float
    → petstore_api: provide it as integer there
      · GET /pets response 200 at "$[].weight"
      · GET /pets/* response 200 at "$.weight" (petstore_api: PetRecord.weight)
`, report)
}

func TestFormatNotDeployableReportPrintsTheEndpointBreaksBeforeTheBlocks(t *testing.T) {
	apiVersion := "4.0.0"
	results := map[string]CanIDeployResult{
		"petstore_api": {
			Deployable:         false,
			ParticipantVersion: &apiVersion,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/pets/*/archive": {
					"post": {
						"200": {
							{Reason: "provider_resource_interaction_not_found", Role: "consumer", Details: map[string]string{"providerStatuses": "204"}},
						},
					},
				},
				"/pets/*": {
					"get": {
						"200": {
							{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.weight", "providerField": "PetRecord.weight", "property": "$.weight",
								"consumerPropertyType": "integer", "providerPropertyType": "float", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
							{Reason: "property_nullable_in_provider_not_nullable_in_consumer", Role: "consumer", Details: map[string]string{
								"property": "$.owner.uuid", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
				},
				"/pets": {
					"get": {
						"200": {
							{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.weight", "providerField": "PetSummary.weight", "property": "$[].weight",
								"consumerPropertyType": "integer", "providerPropertyType": "float", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
				},
			},
		},
	}

	report := FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 2.0.0", "petstore_web", "production", results)

	assert.Equal(t, `petstore_web 2.0.0 cannot be deployed to production

petstore_api (4.0.0, deployed):
  POST /pets/*/archive
    response 200:
      - petstore_web reads response 200, but petstore_api answers only 204
        → petstore_web: read response 204
        → petstore_api: answer 200
  "$.owner.uuid" — petstore_web doesn't accept null, but petstore_api may provide null
    → petstore_web: handle null in the code, then mark it nullable
    → petstore_api: stop providing null
      · GET /pets/* response 200 at "$.owner.uuid"
  Pet.weight — petstore_web reads it as integer, but petstore_api provides float (2 places)
    → petstore_web: read it as float
    → petstore_api: provide it as integer there
      · GET /pets response 200 at "$[].weight" (petstore_api: PetSummary.weight)
      · GET /pets/* response 200 at "$.weight" (petstore_api: PetRecord.weight)
`, report)
}

func TestFormatNotDeployableReportNamesABreakWithoutFieldsByItsProperty(t *testing.T) {
	results := map[string]CanIDeployResult{
		"petstore_api": {
			Deployable: false,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/pets/*": {
					"get": {
						"200": {
							{Reason: "property_nullable_in_provider_not_nullable_in_consumer", Role: "consumer", Details: map[string]string{
								"consumerField": "Pet.tag", "providerField": "PetRecord.tag", "property": "$.tag", "propertyType": "string",
								"consumerName": "petstore_web", "providerName": "petstore_api",
							}},
							{Reason: "property_nullable_in_provider_not_nullable_in_consumer", Role: "consumer", Details: map[string]string{
								"property": "$.owner.uuid", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
							}},
						},
					},
				},
			},
		},
	}

	report := FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 2.0.0", "petstore_web", "production", results)

	assert.Equal(t, `petstore_web 2.0.0 cannot be deployed to production

petstore_api:
  "$.owner.uuid" — petstore_web doesn't accept null, but petstore_api may provide null
    → petstore_web: handle null in the code, then mark it nullable
    → petstore_api: stop providing null
      · GET /pets/* response 200 at "$.owner.uuid"
  Pet.tag — petstore_web doesn't accept null, but petstore_api may provide null
    → petstore_web: handle null in the code, then mark it nullable
    → petstore_api: stop providing null there
      · GET /pets/* response 200 at "$.tag" (petstore_api: PetRecord.tag)
`, report)
}

func TestFormatNotDeployableReportWritesAPropertyBreakAsAFieldBlock(t *testing.T) {
	type checkedPlace struct {
		participant, counterpart, endpoint, method, interaction string
	}
	webGetsPet := checkedPlace{participant: "petstore_web", counterpart: "petstore_api", endpoint: "/pets/*", method: "get", interaction: "200"}
	apiServesPet := checkedPlace{participant: "petstore_api", counterpart: "petstore_web", endpoint: "/pets/*", method: "get", interaction: "200"}
	webPostsPet := checkedPlace{participant: "petstore_web", counterpart: "petstore_api", endpoint: "/pets", method: "post", interaction: "request"}
	apiReceivesPet := checkedPlace{participant: "petstore_api", counterpart: "petstore_web", endpoint: "/pets", method: "post", interaction: "request"}
	spaGetsFiles := checkedPlace{participant: "spa", counterpart: "api", endpoint: "/files", method: "get", interaction: "200"}
	spaGetsSession := checkedPlace{participant: "spa", counterpart: "api", endpoint: "/session", method: "get", interaction: "200"}
	apiServesSession := checkedPlace{participant: "api", counterpart: "spa", endpoint: "/session", method: "get", interaction: "200"}
	spaPostsSession := checkedPlace{participant: "spa", counterpart: "api", endpoint: "/session", method: "post", interaction: "request"}
	apiReceivesSession := checkedPlace{participant: "api", counterpart: "spa", endpoint: "/session", method: "post", interaction: "request"}

	tests := []struct {
		name          string
		place         checkedPlace
		contractBreak ContractBreak
		block         string
	}{
		{
			name:  "property_missing_in_provider as consumer on a response",
			place: webGetsPet,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
				"consumerField": "Pet.photoUrl", "property": "$.photoUrl", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			block: `  Pet.photoUrl — petstore_web reads it, but petstore_api doesn't provide it
    → petstore_web: stop reading it, or handle its absence in the code and mark it optional
    → petstore_api: provide it
      · GET /pets/* response 200 at "$.photoUrl"
`,
		},
		{
			name:  "property_missing_in_provider as provider on a response is named by the consumer's field",
			place: apiServesPet,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "provider", Details: map[string]string{
				"consumerField": "Pet.photoUrl", "property": "$.photoUrl", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			block: `  Pet.photoUrl — petstore_api doesn't provide it, but petstore_web reads it
    → petstore_api: provide it
    → petstore_web: stop reading it, or handle its absence in the code and mark it optional there
      · GET /pets/* response 200 at "$.photoUrl" (petstore_web: Pet.photoUrl)
`,
		},
		{
			name:  "property_optional_in_provider_required_in_consumer as consumer on a response",
			place: webGetsPet,
			contractBreak: ContractBreak{Reason: "property_optional_in_provider_required_in_consumer", Role: "consumer", Details: map[string]string{
				"consumerField": "Pet.status", "providerField": "PetRecord.status", "property": "$.status", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			block: `  Pet.status — petstore_web requires it, but petstore_api only sometimes provides it
    → petstore_web: handle its absence in the code, then mark it optional
    → petstore_api: always provide it there
      · GET /pets/* response 200 at "$.status" (petstore_api: PetRecord.status)
`,
		},
		{
			name:  "property_optional_in_provider_required_in_consumer as provider on a response",
			place: apiServesPet,
			contractBreak: ContractBreak{Reason: "property_optional_in_provider_required_in_consumer", Role: "provider", Details: map[string]string{
				"consumerField": "Pet.status", "providerField": "PetRecord.status", "property": "$.status", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			block: `  PetRecord.status — petstore_api provides it only sometimes, but petstore_web requires it
    → petstore_api: always provide it
    → petstore_web: handle its absence in the code, then mark it optional there
      · GET /pets/* response 200 at "$.status" (petstore_web: Pet.status)
`,
		},
		{
			name:  "property_type_mismatch as consumer on a response",
			place: webGetsPet,
			contractBreak: ContractBreak{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
				"consumerField": "Pet.weight", "providerField": "PetRecord.weight", "property": "$.weight", "consumerPropertyType": "integer", "providerPropertyType": "float",
				"consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			block: `  Pet.weight — petstore_web reads it as integer, but petstore_api provides float
    → petstore_web: read it as float
    → petstore_api: provide it as integer there
      · GET /pets/* response 200 at "$.weight" (petstore_api: PetRecord.weight)
`,
		},
		{
			name:  "property_type_mismatch as provider on a response",
			place: apiServesPet,
			contractBreak: ContractBreak{Reason: "property_type_mismatch", Role: "provider", Details: map[string]string{
				"consumerField": "Pet.weight", "providerField": "PetRecord.weight", "property": "$.weight", "consumerPropertyType": "integer", "providerPropertyType": "float",
				"consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			block: `  PetRecord.weight — petstore_api provides it as float, but petstore_web reads integer
    → petstore_api: provide it as integer
    → petstore_web: read it as float there
      · GET /pets/* response 200 at "$.weight" (petstore_web: Pet.weight)
`,
		},
		{
			name:  "property_nullable_in_provider_not_nullable_in_consumer as consumer on a response",
			place: webGetsPet,
			contractBreak: ContractBreak{Reason: "property_nullable_in_provider_not_nullable_in_consumer", Role: "consumer", Details: map[string]string{
				"consumerField": "Pet.tag", "providerField": "PetRecord.tag", "property": "$.tag", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			block: `  Pet.tag — petstore_web doesn't accept null, but petstore_api may provide null
    → petstore_web: handle null in the code, then mark it nullable
    → petstore_api: stop providing null there
      · GET /pets/* response 200 at "$.tag" (petstore_api: PetRecord.tag)
`,
		},
		{
			name:  "property_nullable_in_provider_not_nullable_in_consumer as provider on a response",
			place: apiServesPet,
			contractBreak: ContractBreak{Reason: "property_nullable_in_provider_not_nullable_in_consumer", Role: "provider", Details: map[string]string{
				"consumerField": "Pet.tag", "providerField": "PetRecord.tag", "property": "$.tag", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			block: `  PetRecord.tag — petstore_api may provide null, but petstore_web doesn't accept it
    → petstore_api: stop providing null
    → petstore_web: handle null in the code, then mark it nullable there
      · GET /pets/* response 200 at "$.tag" (petstore_web: Pet.tag)
`,
		},
		{
			name:  "property_missing_in_consumer as consumer on a request is named by the provider's field",
			place: webPostsPet,
			contractBreak: ContractBreak{Reason: "property_missing_in_consumer", Role: "consumer", Details: map[string]string{
				"providerField": "NewPetRecord.ownerId", "property": "$.ownerId", "propertyType": "integer", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			block: `  NewPetRecord.ownerId — petstore_web doesn't send it, but petstore_api requires it
    → petstore_web: send it
    → petstore_api: stop requiring it, then mark it optional there
      · POST /pets request at "$.ownerId" (petstore_api: NewPetRecord.ownerId)
`,
		},
		{
			name:  "property_missing_in_consumer as provider on a request",
			place: apiReceivesPet,
			contractBreak: ContractBreak{Reason: "property_missing_in_consumer", Role: "provider", Details: map[string]string{
				"providerField": "NewPetRecord.ownerId", "property": "$.ownerId", "propertyType": "integer", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			block: `  NewPetRecord.ownerId — petstore_api requires it, but petstore_web doesn't send it
    → petstore_api: stop requiring it, then mark it optional
    → petstore_web: send it
      · POST /pets request at "$.ownerId"
`,
		},
		{
			name:  "property_optional_in_consumer_required_in_provider as consumer on a request",
			place: webPostsPet,
			contractBreak: ContractBreak{Reason: "property_optional_in_consumer_required_in_provider", Role: "consumer", Details: map[string]string{
				"consumerField": "NewPet.name", "providerField": "NewPetRecord.name", "property": "$.name", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			block: `  NewPet.name — petstore_web sends it only sometimes, but petstore_api requires it
    → petstore_web: always send it
    → petstore_api: handle its absence in the code, then mark it optional there
      · POST /pets request at "$.name" (petstore_api: NewPetRecord.name)
`,
		},
		{
			name:  "property_optional_in_consumer_required_in_provider as provider on a request",
			place: apiReceivesPet,
			contractBreak: ContractBreak{Reason: "property_optional_in_consumer_required_in_provider", Role: "provider", Details: map[string]string{
				"consumerField": "NewPet.name", "providerField": "NewPetRecord.name", "property": "$.name", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			block: `  NewPetRecord.name — petstore_api requires it, but petstore_web sends it only sometimes
    → petstore_api: handle its absence in the code, then mark it optional
    → petstore_web: always send it there
      · POST /pets request at "$.name" (petstore_web: NewPet.name)
`,
		},
		{
			name:  "property_type_mismatch as consumer on a request",
			place: webPostsPet,
			contractBreak: ContractBreak{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
				"consumerField": "NewPet.age", "providerField": "NewPetRecord.age", "property": "$.age", "consumerPropertyType": "string", "providerPropertyType": "integer",
				"consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			block: `  NewPet.age — petstore_web sends it as string, but petstore_api expects integer
    → petstore_web: send integer
    → petstore_api: accept string there
      · POST /pets request at "$.age" (petstore_api: NewPetRecord.age)
`,
		},
		{
			name:  "property_type_mismatch as provider on a request",
			place: apiReceivesPet,
			contractBreak: ContractBreak{Reason: "property_type_mismatch", Role: "provider", Details: map[string]string{
				"consumerField": "NewPet.age", "providerField": "NewPetRecord.age", "property": "$.age", "consumerPropertyType": "string", "providerPropertyType": "integer",
				"consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			block: `  NewPetRecord.age — petstore_api expects it as integer, but petstore_web sends string
    → petstore_api: accept string
    → petstore_web: send integer there
      · POST /pets request at "$.age" (petstore_web: NewPet.age)
`,
		},
		{
			name:  "property_nullable_in_consumer_not_nullable_in_provider as consumer on a request",
			place: webPostsPet,
			contractBreak: ContractBreak{Reason: "property_nullable_in_consumer_not_nullable_in_provider", Role: "consumer", Details: map[string]string{
				"consumerField": "NewPet.birthDate", "providerField": "NewPetRecord.birthDate", "property": "$.birthDate", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			block: `  NewPet.birthDate — petstore_web may send null, but petstore_api doesn't accept it
    → petstore_web: stop sending null
    → petstore_api: handle null in the code, then mark it nullable there
      · POST /pets request at "$.birthDate" (petstore_api: NewPetRecord.birthDate)
`,
		},
		{
			name:  "property_nullable_in_consumer_not_nullable_in_provider as provider on a request",
			place: apiReceivesPet,
			contractBreak: ContractBreak{Reason: "property_nullable_in_consumer_not_nullable_in_provider", Role: "provider", Details: map[string]string{
				"consumerField": "NewPet.birthDate", "providerField": "NewPetRecord.birthDate", "property": "$.birthDate", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			block: `  NewPetRecord.birthDate — petstore_api doesn't accept null, but petstore_web may send it
    → petstore_api: handle null in the code, then mark it nullable
    → petstore_web: stop sending null there
      · POST /pets request at "$.birthDate" (petstore_web: NewPet.birthDate)
`,
		},
		{
			name:  "checked side without a field is named by the other side's field",
			place: webGetsPet,
			contractBreak: ContractBreak{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
				"providerField": "PetRecord.weight", "property": "$.weight", "consumerPropertyType": "integer", "providerPropertyType": "float",
				"consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			block: `  PetRecord.weight — petstore_web reads it as integer, but petstore_api provides float
    → petstore_web: read it as float
    → petstore_api: provide it as integer there
      · GET /pets/* response 200 at "$.weight" (petstore_api: PetRecord.weight)
`,
		},
		{
			name:  "other side without a field ends its fix without there",
			place: webGetsPet,
			contractBreak: ContractBreak{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
				"consumerField": "Pet.weight", "property": "$.weight", "consumerPropertyType": "integer", "providerPropertyType": "float",
				"consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			block: `  Pet.weight — petstore_web reads it as integer, but petstore_api provides float
    → petstore_web: read it as float
    → petstore_api: provide it as integer
      · GET /pets/* response 200 at "$.weight"
`,
		},
		{
			name:  "break without fields is named by its property",
			place: webGetsPet,
			contractBreak: ContractBreak{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
				"property": "$.weight", "consumerPropertyType": "integer", "providerPropertyType": "float", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			block: `  "$.weight" — petstore_web reads it as integer, but petstore_api provides float
    → petstore_web: read it as float
    → petstore_api: provide it as integer
      · GET /pets/* response 200 at "$.weight"
`,
		},
		{
			name:  "property_missing_in_provider at the root path is placed at the body",
			place: spaGetsFiles,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
				"consumerField": "Order", "property": "$", "propertyType": "object", "consumerName": "spa", "providerName": "api",
			}},
			block: `  Order — spa reads it, but api doesn't provide it
    → spa: stop reading it, or handle its absence in the code and mark it optional
    → api: provide it
      · GET /files response 200 at the body
`,
		},
		{
			name:  "property_type_mismatch at the root path is placed at the body",
			place: spaGetsFiles,
			contractBreak: ContractBreak{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
				"consumerField": "Order", "providerField": "Orders", "property": "$", "consumerPropertyType": "object", "providerPropertyType": "array<object>",
				"consumerName": "spa", "providerName": "api",
			}},
			block: `  Order — spa reads it as object, but api provides array<object>
    → spa: read it as array<object>
    → api: provide it as object there
      · GET /files response 200 at the body (api: Orders)
`,
		},
		{
			name:  "break without fields at the root path is named the body",
			place: spaGetsFiles,
			contractBreak: ContractBreak{Reason: "property_type_mismatch", Role: "consumer", Details: map[string]string{
				"property": "$", "consumerPropertyType": "object", "providerPropertyType": "array<object>", "consumerName": "spa", "providerName": "api",
			}},
			block: `  the body — spa reads it as object, but api provides array<object>
    → spa: read it as array<object>
    → api: provide it as object
      · GET /files response 200 at the body
`,
		},
		{
			name:  "property_missing_in_provider at a map's values reads the parent field the way the provider provides it",
			place: spaGetsSession,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
				"consumerField": "Session.onboarding{}", "property": "$.onboarding{}", "propertyType": "boolean", "consumerName": "spa", "providerName": "api",
			}},
			block: `  Session.onboarding{} — spa reads it, but api doesn't provide it
    → spa: read Session.onboarding the way api provides it
    → api: provide it
      · GET /session response 200 at "$.onboarding{}"
`,
		},
		{
			name:  "property_missing_in_provider at a map's values as provider leaves there off the consumer's parent fix",
			place: apiServesSession,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "provider", Details: map[string]string{
				"consumerField": "Session.onboarding{}", "property": "$.onboarding{}", "propertyType": "boolean", "consumerName": "spa", "providerName": "api",
			}},
			block: `  Session.onboarding{} — api doesn't provide it, but spa reads it
    → api: provide it
    → spa: read Session.onboarding the way api provides it
      · GET /session response 200 at "$.onboarding{}" (spa: Session.onboarding{})
`,
		},
		{
			name:  "property_optional_in_provider_required_in_consumer at an array's items reads the parent field the way the provider provides it",
			place: spaGetsSession,
			contractBreak: ContractBreak{Reason: "property_optional_in_provider_required_in_consumer", Role: "consumer", Details: map[string]string{
				"consumerField": "Session.tags[]", "providerField": "SessionRecord.tags[]", "property": "$.tags[]", "propertyType": "string", "consumerName": "spa", "providerName": "api",
			}},
			block: `  Session.tags[] — spa requires it, but api only sometimes provides it
    → spa: read Session.tags the way api provides it
    → api: always provide it there
      · GET /session response 200 at "$.tags[]" (api: SessionRecord.tags[])
`,
		},
		{
			name:  "property_missing_in_provider at the root's items reads the body's field the way the provider provides it",
			place: spaGetsFiles,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
				"consumerField": "Files[]", "property": "$[]", "propertyType": "string", "consumerName": "spa", "providerName": "api",
			}},
			block: `  Files[] — spa reads it, but api doesn't provide it
    → spa: read Files the way api provides it
    → api: provide it
      · GET /files response 200 at "$[]"
`,
		},
		{
			name:  "property_missing_in_consumer at a map's values reads the parent field the way the consumer sends it",
			place: apiReceivesSession,
			contractBreak: ContractBreak{Reason: "property_missing_in_consumer", Role: "provider", Details: map[string]string{
				"providerField": "NewSessionRecord.flags{}", "property": "$.flags{}", "propertyType": "boolean", "consumerName": "spa", "providerName": "api",
			}},
			block: `  NewSessionRecord.flags{} — api requires it, but spa doesn't send it
    → api: read NewSessionRecord.flags the way spa sends it
    → spa: send it
      · POST /session request at "$.flags{}"
`,
		},
		{
			name:  "property_optional_in_consumer_required_in_provider at a map's values as consumer leaves there off the provider's parent fix",
			place: spaPostsSession,
			contractBreak: ContractBreak{Reason: "property_optional_in_consumer_required_in_provider", Role: "consumer", Details: map[string]string{
				"consumerField": "NewSession.flags{}", "providerField": "NewSessionRecord.flags{}", "property": "$.flags{}", "propertyType": "boolean", "consumerName": "spa", "providerName": "api",
			}},
			block: `  NewSession.flags{} — spa sends it only sometimes, but api requires it
    → spa: always send it
    → api: read NewSessionRecord.flags the way spa sends it
      · POST /session request at "$.flags{}" (api: NewSessionRecord.flags{})
`,
		},
		{
			name:  "a fix that doesn't mark it optional keeps its words and there at an array's items",
			place: spaGetsSession,
			contractBreak: ContractBreak{Reason: "property_nullable_in_provider_not_nullable_in_consumer", Role: "consumer", Details: map[string]string{
				"consumerField": "Session.tags[]", "providerField": "SessionRecord.tags[]", "property": "$.tags[]", "propertyType": "string", "consumerName": "spa", "providerName": "api",
			}},
			block: `  Session.tags[] — spa doesn't accept null, but api may provide null
    → spa: handle null in the code, then mark it nullable
    → api: stop providing null there
      · GET /session response 200 at "$.tags[]" (api: SessionRecord.tags[])
`,
		},
		{
			name:  "break without fields at a map's values reads the parent property the way the provider provides it",
			place: spaGetsSession,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
				"property": "$.onboarding{}", "propertyType": "boolean", "consumerName": "spa", "providerName": "api",
			}},
			block: `  "$.onboarding{}" — spa reads it, but api doesn't provide it
    → spa: read "$.onboarding" the way api provides it
    → api: provide it
      · GET /session response 200 at "$.onboarding{}"
`,
		},
		{
			name:  "break without fields at the root's items reads the body the way the provider provides it",
			place: spaGetsFiles,
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
				"property": "$[]", "propertyType": "string", "consumerName": "spa", "providerName": "api",
			}},
			block: `  "$[]" — spa reads it, but api doesn't provide it
    → spa: read the body the way api provides it
    → api: provide it
      · GET /files response 200 at "$[]"
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results := map[string]CanIDeployResult{
				test.place.counterpart: {
					Deployable: false,
					Endpoints: map[string]map[string]map[string][]ContractBreak{
						test.place.endpoint: {test.place.method: {test.place.interaction: {test.contractBreak}}},
					},
				},
			}

			report := FormatNotDeployableReport(paint.For(&bytes.Buffer{}), test.place.participant+" 2.0.0", test.place.participant, "production", results)

			assert.Equal(t, test.place.participant+" 2.0.0 cannot be deployed to production\n\n"+test.place.counterpart+":\n"+test.block, report)
		})
	}
}

func TestFormatNotDeployableReportKeepsABreakItCannotNameUnderItsEndpoint(t *testing.T) {
	tests := []struct {
		name                          string
		endpoint, method, interaction string
		contractBreak                 ContractBreak
		endpointBreaks                string
	}{
		{
			name:     "unknown reason with a property",
			endpoint: "/pets/*", method: "get", interaction: "200",
			contractBreak: ContractBreak{Reason: "some_future_reason", Role: "consumer", Details: map[string]string{
				"consumerField": "Pet.photoUrl", "property": "$.photoUrl", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			endpointBreaks: `  GET /pets/*
    response 200:
      - some_future_reason (consumerField: Pet.photoUrl, consumerName: petstore_web, property: $.photoUrl, providerName: petstore_api)
`,
		},
		{
			name:     "missing role falls back to the raw line instead of inferring it from consumerName",
			endpoint: "/pets/*", method: "get", interaction: "200",
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Details: map[string]string{
				"property": "$.photoUrl", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			endpointBreaks: `  GET /pets/*
    response 200:
      - property_missing_in_provider (consumerName: petstore_web, property: $.photoUrl, propertyType: string, providerName: petstore_api)
`,
		},
		{
			name:     "unknown role falls back to the raw line",
			endpoint: "/pets/*", method: "get", interaction: "200",
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "observer", Details: map[string]string{
				"property": "$.photoUrl", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			endpointBreaks: `  GET /pets/*
    response 200:
      - property_missing_in_provider (consumerName: petstore_web, property: $.photoUrl, propertyType: string, providerName: petstore_api)
`,
		},
		{
			name:     "fallback keeps the raw root path",
			endpoint: "/files", method: "get", interaction: "200",
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "observer", Details: map[string]string{
				"property": "$", "propertyType": "object", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			endpointBreaks: `  GET /files
    response 200:
      - property_missing_in_provider (consumerName: petstore_web, property: $, propertyType: object, providerName: petstore_api)
`,
		},
		{
			name:     "fallback keeps the raw fields",
			endpoint: "/pets/*", method: "get", interaction: "200",
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "observer", Details: map[string]string{
				"property": "$.photoUrl", "consumerField": "Pet.photoUrl", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			endpointBreaks: `  GET /pets/*
    response 200:
      - property_missing_in_provider (consumerField: Pet.photoUrl, consumerName: petstore_web, property: $.photoUrl, providerName: petstore_api)
`,
		},
		{
			name:     "property_missing_in_provider on a request is outside the catalog and falls back to the raw line",
			endpoint: "/pets", method: "post", interaction: "request",
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
				"property": "$.name", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			endpointBreaks: `  POST /pets
    request:
      - property_missing_in_provider (consumerName: petstore_web, property: $.name, propertyType: string, providerName: petstore_api)
`,
		},
		{
			name:     "property break without its property falls back to the raw line",
			endpoint: "/pets/*", method: "get", interaction: "200",
			contractBreak: ContractBreak{Reason: "property_missing_in_provider", Role: "consumer", Details: map[string]string{
				"consumerField": "Pet.photoUrl", "propertyType": "string", "consumerName": "petstore_web", "providerName": "petstore_api",
			}},
			endpointBreaks: `  GET /pets/*
    response 200:
      - property_missing_in_provider (consumerField: Pet.photoUrl, consumerName: petstore_web, propertyType: string, providerName: petstore_api)
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results := map[string]CanIDeployResult{
				"petstore_api": {
					Deployable: false,
					Endpoints: map[string]map[string]map[string][]ContractBreak{
						test.endpoint: {test.method: {test.interaction: {test.contractBreak}}},
					},
				},
			}

			report := FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 2.0.0", "petstore_web", "production", results)

			assert.Equal(t, "petstore_web 2.0.0 cannot be deployed to production\n\npetstore_api:\n"+test.endpointBreaks, report)
		})
	}
}

func TestFormatBreakLine(t *testing.T) {
	webGetsPet := checkedInteraction{participant: "petstore_web", counterpart: "petstore_api", environment: "production", method: "GET", endpoint: "/pets/*"}
	apiServesPet := checkedInteraction{participant: "petstore_api", counterpart: "petstore_web", environment: "production", method: "GET", endpoint: "/pets/*"}
	webGetsReview := checkedInteraction{participant: "petstore_web", counterpart: "petstore_reviews", environment: "production", method: "GET", endpoint: "/reviews/*"}
	siteReadsUnsubscribe200 := checkedInteraction{participant: "site", counterpart: "api", environment: "production", method: "POST", endpoint: "/api/newsletter/unsubscribe", status: "200"}
	siteReadsUpload201 := checkedInteraction{participant: "site", counterpart: "api", environment: "production", method: "POST", endpoint: "/api/files", status: "201"}
	appSendsRestore := checkedInteraction{participant: "app", counterpart: "api", environment: "production", method: "POST", endpoint: "/api/users/*/restore", isRequest: true}

	tests := []struct {
		name          string
		interaction   checkedInteraction
		contractBreak ContractBreak
		line          breakLine
	}{
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
			name:        "provider_resource_interaction_not_found as consumer on a response names the status the provider answers",
			interaction: siteReadsUnsubscribe200,
			contractBreak: ContractBreak{Reason: "provider_resource_interaction_not_found", Role: "consumer", Details: map[string]string{
				"providerStatuses": "204",
			}},
			line: breakLine{
				text: "site reads response 200, but api answers only 204",
				fixes: []string{
					"site: read response 204",
					"api: answer 200",
				},
			},
		},
		{
			name:        "provider_resource_interaction_not_found as consumer on a response joins two statuses with or",
			interaction: siteReadsUnsubscribe200,
			contractBreak: ContractBreak{Reason: "provider_resource_interaction_not_found", Role: "consumer", Details: map[string]string{
				"providerStatuses": "201, 204",
			}},
			line: breakLine{
				text: "site reads response 200, but api answers only 201 or 204",
				fixes: []string{
					"site: read response 201 or 204",
					"api: answer 200",
				},
			},
		},
		{
			name:        "provider_resource_interaction_not_found as consumer on a response puts or before the last of several statuses",
			interaction: siteReadsUnsubscribe200,
			contractBreak: ContractBreak{Reason: "provider_resource_interaction_not_found", Role: "consumer", Details: map[string]string{
				"providerStatuses": "201, 204, 404",
			}},
			line: breakLine{
				text: "site reads response 200, but api answers only 201, 204 or 404",
				fixes: []string{
					"site: read response 201, 204 or 404",
					"api: answer 200",
				},
			},
		},
		{
			name:          "provider_resource_interaction_not_found as consumer on a response the provider declares none of",
			interaction:   siteReadsUpload201,
			contractBreak: ContractBreak{Reason: "provider_resource_interaction_not_found", Role: "consumer"},
			line: breakLine{
				text: "site reads response 201, but api declares no response",
				fixes: []string{
					"site: stop reading it",
					"api: answer 201",
				},
			},
		},
		{
			name:        "provider_resource_interaction_not_found as consumer on a request ignores the statuses the provider answers",
			interaction: appSendsRestore,
			contractBreak: ContractBreak{Reason: "provider_resource_interaction_not_found", Role: "consumer", Details: map[string]string{
				"providerStatuses": "204, 404",
			}},
			line: breakLine{
				text: "app sends a request body, but api declares none",
				fixes: []string{
					"app: stop sending it",
					"api: accept a request body",
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
			name:          "unknown reason falls back to the bare reason",
			interaction:   webGetsPet,
			contractBreak: ContractBreak{Reason: "some_future_reason", Role: "consumer"},
			line:          breakLine{text: `some_future_reason`},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.line, formatBreakLine(test.interaction, test.contractBreak))
		})
	}
}
