package validate_contract

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bidirekt/cli/internal/components"
	"github.com/bidirekt/cli/internal/reports"
)

type ValidateContractClient struct {
	httpClient *components.HTTPClient
}

func NewValidateContractClient(httpClient *components.HTTPClient) *ValidateContractClient {
	return &ValidateContractClient{httpClient: httpClient}
}

func (this *ValidateContractClient) ValidateContract(ctx context.Context, requestBody *ValidateContractRequestBody) (ValidateContractResponseBody, error) {
	requestBodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return ValidateContractResponseBody{}, fmt.Errorf("cannot serialize contract to JSON: %w", err)
	}

	response, err := this.httpClient.Post(ctx, "/api/contracts/validate", requestBodyJSON)
	if err != nil {
		return ValidateContractResponseBody{}, fmt.Errorf("cannot validate contract on broker: %w", err)
	}

	var responseBody ValidateContractResponseBody
	if err := json.Unmarshal(response.Bytes(), &responseBody); err != nil {
		return ValidateContractResponseBody{}, fmt.Errorf("cannot parse validate response: %w", err)
	}

	if response.StatusCode() != http.StatusOK {
		if len(responseBody.Violations) > 0 {
			return ValidateContractResponseBody{}, &reports.ValidationFailedError{Message: responseBody.Message, Violations: responseBody.Violations}
		}

		return ValidateContractResponseBody{}, fmt.Errorf("cannot validate contract on broker: %s", responseBody.Message)
	}

	return responseBody, nil
}
