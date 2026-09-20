package create_environment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bidirekt/cli/internal/components"
)

type CreateEnvironmentClient struct {
	httpClient *components.HTTPClient
}

func NewCreateEnvironmentClient(httpClient *components.HTTPClient) *CreateEnvironmentClient {
	return &CreateEnvironmentClient{httpClient: httpClient}
}

func (c *CreateEnvironmentClient) Create(
	ctx context.Context,
	requestBody *CreateEnvironmentRequestBody,
) (string, error) {
	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("cannot serialize environment to JSON: %w", err)
	}

	response, err := c.httpClient.Post(ctx, "/api/environments", bodyJSON)
	if err != nil {
		return "", fmt.Errorf("cannot post environment to broker: %w", err)
	}

	var responseBody CreateEnvironmentResponseBody
	if err := json.Unmarshal(response.Bytes(), &responseBody); err != nil {
		return "", fmt.Errorf("cannot parse environment response: %w", err)
	}

	if response.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("cannot post environment to broker: %s", responseBody.Message)
	}

	return responseBody.Message, nil
}
