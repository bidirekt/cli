package components

import (
	"context"
	"net/http"

	"resty.dev/v3"
)

type HTTPClient struct {
	restClient *resty.Client
}

func NewHTTPClient() *HTTPClient {
	httpClient := &HTTPClient{
		restClient: resty.New(),
	}

	httpClient.
		restClient.
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")

	return httpClient
}

func (this *HTTPClient) StdClient() *http.Client {
	return this.restClient.Client()
}

func (this *HTTPClient) UseBroker(brokerURL, apiKey string) {
	this.restClient.SetBaseURL(brokerURL)
	if apiKey != "" {
		this.restClient.SetHeader("Authorization", "Bearer "+apiKey)
	}
}

func (this *HTTPClient) ExplainAuthFailures(brokerURL string) {
	broker := RedactedURL(brokerURL)
	this.restClient.AddResponseMiddleware(func(_ *resty.Client, response *resty.Response) error {
		switch response.StatusCode() {
		case http.StatusUnauthorized:
			return NotSignedInError{BrokerURL: broker}
		case http.StatusForbidden:
			return ForbiddenError{BrokerURL: broker}
		default:
			return nil
		}
	})
}

func (this *HTTPClient) Get(ctx context.Context, url string) (*resty.Response, error) {
	return this.restClient.R().SetContext(ctx).Get(url)
}

func (this *HTTPClient) Post(ctx context.Context, url string, body any) (*resty.Response, error) {
	return this.restClient.R().SetContext(ctx).SetBody(body).Post(url)
}
