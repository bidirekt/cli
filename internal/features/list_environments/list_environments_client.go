package list_environments

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bidirekt/cli/internal/components"
)

type ListEnvironmentsClient struct {
	httpClient *components.HTTPClient
}

func NewListEnvironmentsClient(httpClient *components.HTTPClient) *ListEnvironmentsClient {
	return &ListEnvironmentsClient{httpClient: httpClient}
}

func (this *ListEnvironmentsClient) List(ctx context.Context) ([]string, error) {
	response, err := this.httpClient.Get(ctx, "/api/environments")
	if err != nil {
		return nil, fmt.Errorf("cannot list environments from broker: %w", err)
	}

	var responseBody ListEnvironmentsResponseBody
	if err := json.Unmarshal(response.Bytes(), &responseBody); err != nil {
		return nil, fmt.Errorf("cannot parse environments response: %w", err)
	}

	if response.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("cannot list environments from broker: %s", responseBody.Message)
	}

	return responseBody.Environments, nil
}
