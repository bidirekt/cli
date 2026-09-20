package create_participant

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bidirekt/cli/internal/components"
)

type CreateParticipantClient struct {
	httpClient *components.HTTPClient
}

func NewCreateParticipantClient(httpClient *components.HTTPClient) *CreateParticipantClient {
	return &CreateParticipantClient{httpClient: httpClient}
}

func (c *CreateParticipantClient) Create(ctx context.Context, requestBody *CreateParticipantRequestBody) (string, error) {
	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("cannot serialize participant to JSON: %w", err)
	}

	response, err := c.httpClient.Post(ctx, "/api/participants", bodyJSON)
	if err != nil {
		return "", fmt.Errorf("cannot post participant to broker: %w", err)
	}

	var responseBody CreateParticipantResponseBody
	if err := json.Unmarshal(response.Bytes(), &responseBody); err != nil {
		return "", fmt.Errorf("cannot parse participant response: %w", err)
	}

	if response.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("cannot post participant to broker: %s", responseBody.Message)
	}

	return responseBody.Message, nil
}
