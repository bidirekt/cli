package components_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bidirekt/cli/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClientAbortsTheRequestWhenItsContextExpires(t *testing.T) {
	const (
		requestTimeout = 50 * time.Millisecond
		abortDeadline  = 3 * time.Second
	)

	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	t.Cleanup(server.Close)
	// registered after server.Close so it runs first: Close waits for the handler to return
	t.Cleanup(func() { close(release) })

	client := components.NewHTTPClient()
	client.UseBroker(server.URL, "")
	body := map[string]string{"participant": "orders"}

	tests := []struct {
		name string
		call func(ctx context.Context) error
	}{
		{name: "Get", call: func(ctx context.Context) error { _, err := client.Get(ctx, "/slow"); return err }},
		{name: "Post", call: func(ctx context.Context) error { _, err := client.Post(ctx, "/slow", body); return err }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			defer cancel()

			result := make(chan error, 1)
			go func() { result <- test.call(ctx) }()

			select {
			case err := <-result:
				require.Error(t, err)
				assert.True(t, errors.Is(err, context.DeadlineExceeded), "expected context.DeadlineExceeded, got: %v", err)
			case <-time.After(abortDeadline):
				t.Fatal("request was not aborted by its context")
			}
		})
	}
}

func TestHTTPClientUseBrokerSendsTheKeyAsABearerToken(t *testing.T) {
	tests := []struct {
		name          string
		apiKey        string
		authorization string
	}{
		{name: "with a key", apiKey: "bdk_acme", authorization: "Bearer bdk_acme"},
		{name: "without a key", apiKey: "", authorization: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authorizations := make(chan string, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authorizations <- r.Header.Get("Authorization")
			}))
			t.Cleanup(server.Close)

			client := components.NewHTTPClient()
			client.UseBroker(server.URL, test.apiKey)
			_, err := client.Get(context.Background(), "/api/participants")

			require.NoError(t, err)
			assert.Equal(t, test.authorization, <-authorizations)
		})
	}
}

func TestHTTPClientExplainAuthFailuresTurnsA401AndA403IntoTheirErrors(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		brokerURL string
		err       error
		text      string
	}{
		{
			name:      "a 401",
			status:    http.StatusUnauthorized,
			brokerURL: "https://broker.acme.internal",
			err:       components.NotSignedInError{BrokerURL: "https://broker.acme.internal"},
			text:      `not signed in to https://broker.acme.internal — run "bidirekt login" or set BIDIREKT_BROKER_URL and BIDIREKT_API_KEY`,
		},
		{
			name:      "a 403",
			status:    http.StatusForbidden,
			brokerURL: "https://broker.acme.internal",
			err:       components.ForbiddenError{BrokerURL: "https://broker.acme.internal"},
			text:      "the API key can't do this — check its permission at https://broker.acme.internal/profile",
		},
		{
			name:      "a 403 from a URL with a trailing slash",
			status:    http.StatusForbidden,
			brokerURL: "https://broker.acme.internal/",
			err:       components.ForbiddenError{BrokerURL: "https://broker.acme.internal/"},
			text:      "the API key can't do this — check its permission at https://broker.acme.internal/profile",
		},
		{
			name:      "a 401 with the password hidden",
			status:    http.StatusUnauthorized,
			brokerURL: "https://ci:segredo@broker.acme.internal",
			err:       components.NotSignedInError{BrokerURL: "https://ci:xxxxx@broker.acme.internal"},
			text:      `not signed in to https://ci:xxxxx@broker.acme.internal — run "bidirekt login" or set BIDIREKT_BROKER_URL and BIDIREKT_API_KEY`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(`{"message":"refused"}`))
			}))
			t.Cleanup(server.Close)

			client := components.NewHTTPClient()
			client.UseBroker(server.URL, "bdk_acme")
			client.ExplainAuthFailures(test.brokerURL)
			_, err := client.Get(context.Background(), "/api/participants")

			require.Error(t, err)
			assert.ErrorIs(t, err, test.err)
			assert.Equal(t, test.text, test.err.Error())
		})
	}
}

func TestHTTPClientLeavesOtherStatusesAndUnexplainedClientsAlone(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		explain bool
	}{
		{name: "a 404 with ExplainAuthFailures", status: http.StatusNotFound, explain: true},
		{name: "a 401 without ExplainAuthFailures", status: http.StatusUnauthorized},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
			}))
			t.Cleanup(server.Close)

			client := components.NewHTTPClient()
			client.UseBroker(server.URL, "bdk_acme")
			if test.explain {
				client.ExplainAuthFailures(server.URL)
			}
			response, err := client.Get(context.Background(), "/api/participants")

			require.NoError(t, err)
			assert.Equal(t, test.status, response.StatusCode())
		})
	}
}
