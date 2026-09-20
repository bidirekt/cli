package can_i_deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bidirekt/cli/internal/components"
)

type CanIDeployClient struct {
	httpClient *components.HTTPClient
}

func NewCanIDeployClient(httpClient *components.HTTPClient) *CanIDeployClient {
	return &CanIDeployClient{httpClient: httpClient}
}

func (c *CanIDeployClient) Check(ctx context.Context, requestBody *CanIDeployRequestBody) (CanIDeployResponseBody, error) {
	body, err := json.Marshal(requestBody)
	if err != nil {
		return CanIDeployResponseBody{}, fmt.Errorf("cannot serialize can-i-deploy request to JSON: %w", err)
	}

	resp, err := c.httpClient.Post(ctx, "/api/can-i-deploy", body)
	if err != nil {
		return CanIDeployResponseBody{}, fmt.Errorf("cannot query can-i-deploy from broker: %w", err)
	}

	var result CanIDeployResponseBody
	if err := json.Unmarshal(resp.Bytes(), &result); err != nil {
		return CanIDeployResponseBody{}, fmt.Errorf("cannot parse can-i-deploy response: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return CanIDeployResponseBody{}, fmt.Errorf("cannot query can-i-deploy from broker: %s", result.Message)
	}

	return result, nil
}
