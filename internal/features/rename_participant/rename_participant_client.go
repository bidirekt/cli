package rename_participant

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bidirekt/cli/internal/components"
)

type RenameParticipantClient struct {
	httpClient *components.HTTPClient
}

func NewRenameParticipantClient(httpClient *components.HTTPClient) *RenameParticipantClient {
	return &RenameParticipantClient{httpClient: httpClient}
}

func (c *RenameParticipantClient) Rename(ctx context.Context, requestBody *RenameParticipantRequestBody) (string, error) {
	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("cannot serialize participant rename to JSON: %w", err)
	}

	resp, err := c.httpClient.Post(ctx, "/api/participants/rename", body)
	if err != nil {
		return "", fmt.Errorf("cannot post participant rename to broker: %w", err)
	}

	var result RenameParticipantResponseBody
	if err := json.Unmarshal(resp.Bytes(), &result); err != nil {
		return "", fmt.Errorf("cannot parse participant rename response: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("cannot post participant rename to broker: %s", result.Message)
	}

	return result.Message, nil
}
