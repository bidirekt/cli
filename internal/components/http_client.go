package components

import (
	"context"
	"net/http"

	"resty.dev/v3"
)

type HTTPClient struct {
	restClient *resty.Client
}

func NewHTTPClient(config *Config) *HTTPClient {
	brokerURL := config.BrokerURL

	if brokerURL == "" {
		brokerURL = "http://localhost:8080"
	}

	httpClient := &HTTPClient{
		restClient: resty.New().SetBaseURL(brokerURL),
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

func (this *HTTPClient) SetBaseURL(url string) {
	this.restClient.SetBaseURL(url)
}

func (this *HTTPClient) Get(ctx context.Context, url string) (*resty.Response, error) {
	return this.restClient.R().SetContext(ctx).Get(url)
}

func (this *HTTPClient) Post(ctx context.Context, url string, body any) (*resty.Response, error) {
	return this.restClient.R().SetContext(ctx).SetBody(body).Post(url)
}

func (this *HTTPClient) Put(ctx context.Context, url string, body any) (*resty.Response, error) {
	return this.restClient.R().SetContext(ctx).SetBody(body).Put(url)
}

func (this *HTTPClient) Delete(ctx context.Context, url string) (*resty.Response, error) {
	return this.restClient.R().SetContext(ctx).Delete(url)
}
