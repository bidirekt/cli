package login

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/bidirekt/cli/internal/components"
)

var errKeyRefused = errors.New("the broker refused this API key")

type LoginClient struct {
	httpClient *components.HTTPClient
}

func NewLoginClient(httpClient *components.HTTPClient) *LoginClient {
	return &LoginClient{httpClient: httpClient}
}

func (this *LoginClient) CheckKey(ctx context.Context, brokerURL, apiKey string) error {
	this.httpClient.UseBroker(brokerURL, apiKey)
	broker := components.RedactedURL(brokerURL)

	response, err := this.httpClient.Get(ctx, "/api/environments")
	if err != nil {
		return fmt.Errorf("cannot reach the broker at %s: %w", broker, err)
	}

	switch response.StatusCode() {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return errKeyRefused
	default:
		return fmt.Errorf("cannot reach the broker at %s: HTTP %d", broker, response.StatusCode())
	}

	var responseBody EnvironmentsResponseBody
	if err := json.Unmarshal(response.Bytes(), &responseBody); err != nil {
		return fmt.Errorf("cannot reach the broker at %s: not a Bidirekt broker", broker)
	}

	return nil
}
