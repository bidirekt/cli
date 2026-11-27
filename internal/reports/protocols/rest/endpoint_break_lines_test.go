package rest

import (
	"testing"

	"github.com/bidirekt/cli/internal/reports"
	"github.com/stretchr/testify/assert"
)

func TestFormatBreakLine(t *testing.T) {
	webGetsPet := checkedInteraction{participant: "petstore_web", counterpart: "petstore_api", environment: "production", method: "GET", endpoint: "/pets/*"}
	apiServesPet := checkedInteraction{participant: "petstore_api", counterpart: "petstore_web", environment: "production", method: "GET", endpoint: "/pets/*"}
	webGetsReview := checkedInteraction{participant: "petstore_web", counterpart: "petstore_reviews", environment: "production", method: "GET", endpoint: "/reviews/*"}
	siteReadsUnsubscribe200 := checkedInteraction{participant: "site", counterpart: "api", environment: "production", method: "POST", endpoint: "/api/newsletter/unsubscribe", status: "200"}
	siteReadsUpload201 := checkedInteraction{participant: "site", counterpart: "api", environment: "production", method: "POST", endpoint: "/api/files", status: "201"}
	appSendsRestore := checkedInteraction{participant: "app", counterpart: "api", environment: "production", method: "POST", endpoint: "/api/users/*/restore", isRequest: true}
	apiAnswersDelete200 := checkedInteraction{participant: "petstore_api", counterpart: "petstore_web", environment: "production", method: "DELETE", endpoint: "/pets/*", status: "200"}
	apiAcceptsCreate := checkedInteraction{participant: "petstore_api", counterpart: "petstore_web", environment: "production", method: "POST", endpoint: "/pets", isRequest: true}

	tests := []struct {
		name          string
		interaction   checkedInteraction
		contractBreak reports.ContractBreak
		line          reports.BreakLine
	}{
		{
			name:          "provider_resource_not_found as consumer",
			interaction:   webGetsReview,
			contractBreak: reports.ContractBreak{Reason: "provider_resource_not_found", Role: "consumer"},
			line: reports.BreakLine{
				Text: "petstore_web calls GET /reviews/*, but petstore_reviews doesn't provide it",
				Fixes: []string{
					"petstore_web: stop calling it",
					"petstore_reviews: publish it",
				},
			},
		},
		{
			name:        "provider_resource_interaction_not_found as consumer on a response names the status the provider answers",
			interaction: siteReadsUnsubscribe200,
			contractBreak: reports.ContractBreak{Reason: "provider_resource_interaction_not_found", Role: "consumer", Details: map[string]string{
				"providerStatuses": "204",
			}},
			line: reports.BreakLine{
				Text: "site reads response 200, but api answers only 204",
				Fixes: []string{
					"site: read response 204",
					"api: answer 200",
				},
			},
		},
		{
			name:        "provider_resource_interaction_not_found as consumer on a response joins two statuses with or",
			interaction: siteReadsUnsubscribe200,
			contractBreak: reports.ContractBreak{Reason: "provider_resource_interaction_not_found", Role: "consumer", Details: map[string]string{
				"providerStatuses": "201, 204",
			}},
			line: reports.BreakLine{
				Text: "site reads response 200, but api answers only 201 or 204",
				Fixes: []string{
					"site: read response 201 or 204",
					"api: answer 200",
				},
			},
		},
		{
			name:        "provider_resource_interaction_not_found as consumer on a response puts or before the last of several statuses",
			interaction: siteReadsUnsubscribe200,
			contractBreak: reports.ContractBreak{Reason: "provider_resource_interaction_not_found", Role: "consumer", Details: map[string]string{
				"providerStatuses": "201, 204, 404",
			}},
			line: reports.BreakLine{
				Text: "site reads response 200, but api answers only 201, 204 or 404",
				Fixes: []string{
					"site: read response 201, 204 or 404",
					"api: answer 200",
				},
			},
		},
		{
			name:          "provider_resource_interaction_not_found as consumer on a response the provider declares none of",
			interaction:   siteReadsUpload201,
			contractBreak: reports.ContractBreak{Reason: "provider_resource_interaction_not_found", Role: "consumer"},
			line: reports.BreakLine{
				Text: "site reads response 201, but api declares no response",
				Fixes: []string{
					"site: stop reading it",
					"api: answer 201",
				},
			},
		},
		{
			name:        "provider_resource_interaction_not_found as consumer on a request ignores the statuses the provider answers",
			interaction: appSendsRestore,
			contractBreak: reports.ContractBreak{Reason: "provider_resource_interaction_not_found", Role: "consumer", Details: map[string]string{
				"providerStatuses": "204, 404",
			}},
			line: reports.BreakLine{
				Text: "app sends a request body, but api declares none",
				Fixes: []string{
					"app: stop sending it",
					"api: accept a request body",
				},
			},
		},
		{
			name:          "provider_resource_not_deployed_in_environment as consumer",
			interaction:   webGetsReview,
			contractBreak: reports.ContractBreak{Reason: "provider_resource_not_deployed_in_environment", Role: "consumer"},
			line: reports.BreakLine{
				Text:  "petstore_web calls GET /reviews/*, but petstore_reviews is not deployed in production",
				Fixes: []string{"petstore_reviews: deploy it to production first"},
			},
		},
		{
			name:          "provider_resource_removed_but_still_consumed as provider",
			interaction:   apiServesPet,
			contractBreak: reports.ContractBreak{Reason: "provider_resource_removed_but_still_consumed", Role: "provider"},
			line: reports.BreakLine{
				Text: "petstore_api removed GET /pets/*, but petstore_web still calls it",
				Fixes: []string{
					"petstore_api: keep it until petstore_web stops calling it",
					"petstore_web: stop calling it",
				},
			},
		},
		{
			name:        "provider_resource_interaction_removed_but_still_consumed as provider on a response names the statuses it now answers",
			interaction: apiAnswersDelete200,
			contractBreak: reports.ContractBreak{Reason: "provider_resource_interaction_removed_but_still_consumed", Role: "provider", Details: map[string]string{
				"providerStatuses": "204",
			}},
			line: reports.BreakLine{
				Text: "petstore_api now answers DELETE /pets/* only with 204, but petstore_web still reads response 200",
				Fixes: []string{
					"petstore_api: keep answering 200 until petstore_web stops reading it",
					"petstore_web: read response 204",
				},
			},
		},
		{
			name:        "provider_resource_interaction_removed_but_still_consumed as provider on a response puts or before the last of several statuses",
			interaction: apiAnswersDelete200,
			contractBreak: reports.ContractBreak{Reason: "provider_resource_interaction_removed_but_still_consumed", Role: "provider", Details: map[string]string{
				"providerStatuses": "202, 204, 404",
			}},
			line: reports.BreakLine{
				Text: "petstore_api now answers DELETE /pets/* only with 202, 204 or 404, but petstore_web still reads response 200",
				Fixes: []string{
					"petstore_api: keep answering 200 until petstore_web stops reading it",
					"petstore_web: read response 202, 204 or 404",
				},
			},
		},
		{
			name:          "provider_resource_interaction_removed_but_still_consumed as provider on a response when no response is left",
			interaction:   apiAnswersDelete200,
			contractBreak: reports.ContractBreak{Reason: "provider_resource_interaction_removed_but_still_consumed", Role: "provider"},
			line: reports.BreakLine{
				Text: "petstore_api no longer answers DELETE /pets/*, but petstore_web still reads response 200",
				Fixes: []string{
					"petstore_api: keep answering 200 until petstore_web stops reading it",
					"petstore_web: stop reading it",
				},
			},
		},
		{
			name:          "provider_resource_interaction_removed_but_still_consumed as provider on a request",
			interaction:   apiAcceptsCreate,
			contractBreak: reports.ContractBreak{Reason: "provider_resource_interaction_removed_but_still_consumed", Role: "provider"},
			line: reports.BreakLine{
				Text: "petstore_api dropped the request body of POST /pets, but petstore_web still sends it",
				Fixes: []string{
					"petstore_api: keep accepting it until petstore_web stops sending it",
					"petstore_web: stop sending it",
				},
			},
		},
		{
			name:          "unknown reason falls back to the bare reason",
			interaction:   webGetsPet,
			contractBreak: reports.ContractBreak{Reason: "some_future_reason", Role: "consumer"},
			line:          reports.BreakLine{Text: `some_future_reason`},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.line, formatBreakLine(test.interaction, test.contractBreak))
		})
	}
}
