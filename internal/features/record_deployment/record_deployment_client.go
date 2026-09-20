package record_deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bidirekt/cli/internal/components"
)

type RecordDeploymentClient struct {
	httpClient *components.HTTPClient
}

func NewRecordDeploymentClient(httpClient *components.HTTPClient) *RecordDeploymentClient {
	return &RecordDeploymentClient{httpClient: httpClient}
}

func (c *RecordDeploymentClient) Record(ctx context.Context, requestBody *RecordDeploymentRequestBody) (string, error) {
	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("cannot serialize deployment to JSON: %w", err)
	}

	response, err := c.httpClient.Post(ctx, "/api/deployments", bodyJSON)
	if err != nil {
		return "", fmt.Errorf("cannot post deployment to broker: %w", err)
	}

	var responseBody RecordDeploymentResponseBody
	if err := json.Unmarshal(response.Bytes(), &responseBody); err != nil {
		return "", fmt.Errorf("cannot parse deployment response: %w", err)
	}

	if response.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("cannot post deployment to broker: %s", responseBody.Message)
	}

	return responseBody.Message, nil
}
