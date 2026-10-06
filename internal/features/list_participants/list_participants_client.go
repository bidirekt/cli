package list_participants

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bidirekt/cli/internal/components"
)

type ListParticipantsClient struct {
	httpClient *components.HTTPClient
}

func NewListParticipantsClient(httpClient *components.HTTPClient) *ListParticipantsClient {
	return &ListParticipantsClient{httpClient: httpClient}
}

func (this *ListParticipantsClient) List(ctx context.Context) ([]string, error) {
	response, err := this.httpClient.Get(ctx, "/api/participants")
	if err != nil {
		return nil, fmt.Errorf("cannot list participants from broker: %w", err)
	}

	var responseBody ListParticipantsResponseBody
	if err := json.Unmarshal(response.Bytes(), &responseBody); err != nil {
		return nil, fmt.Errorf("cannot parse participants response: %w", err)
	}

	if response.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("cannot list participants from broker: %s", responseBody.Message)
	}

	return responseBody.Participants, nil
}
