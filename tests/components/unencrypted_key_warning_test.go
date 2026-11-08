package components_test

import (
	"bytes"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWarnIfKeyTravelsUnencrypted(t *testing.T) {
	tests := []struct {
		name      string
		brokerURL string
		apiKey    string
		stderr    string
	}{
		{name: "a key over http to a remote host", brokerURL: "http://broker:8080", apiKey: "bdk_acme", stderr: "warning: the API key travels unencrypted to http://broker:8080\n"},
		{name: "with the password hidden", brokerURL: "http://ci:segredo@broker:8080", apiKey: "bdk_acme", stderr: "warning: the API key travels unencrypted to http://ci:xxxxx@broker:8080\n"},
		{name: "a key over http to localhost", brokerURL: "http://localhost:8080", apiKey: "bdk_acme"},
		{name: "a key over http to 127.0.0.1", brokerURL: "http://127.0.0.1", apiKey: "bdk_acme"},
		{name: "a key over https", brokerURL: "https://broker.acme.internal", apiKey: "bdk_acme"},
		{name: "no key over http to a remote host", brokerURL: "http://broker:8080"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stderr bytes.Buffer

			err := components.WarnIfKeyTravelsUnencrypted(&stderr, test.brokerURL, test.apiKey)

			require.NoError(t, err)
			assert.Equal(t, test.stderr, stderr.String())
		})
	}
}
