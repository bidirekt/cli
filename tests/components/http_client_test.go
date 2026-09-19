package components_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/contracttesting/cli/internal/components"
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

	client := components.NewHTTPClient(&components.Config{BrokerURL: server.URL})
	body := map[string]string{"participant": "orders"}

	tests := []struct {
		name string
		call func(ctx context.Context) error
	}{
		{name: "Get", call: func(ctx context.Context) error { _, err := client.Get(ctx, "/slow"); return err }},
		{name: "Post", call: func(ctx context.Context) error { _, err := client.Post(ctx, "/slow", body); return err }},
		{name: "Put", call: func(ctx context.Context) error { _, err := client.Put(ctx, "/slow", body); return err }},
		{name: "Delete", call: func(ctx context.Context) error { _, err := client.Delete(ctx, "/slow"); return err }},
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
