package components_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptBrokerURL(t *testing.T) {
	tests := []struct {
		name      string
		current   string
		stdin     string
		brokerURL string
		stderr    string
	}{
		{
			name:      "asks without a current value",
			stdin:     "https://broker.acme.internal\n",
			brokerURL: "https://broker.acme.internal",
			stderr:    "Broker URL: ",
		},
		{
			name:      "an answer without a trailing newline counts",
			stdin:     "https://broker.acme.internal",
			brokerURL: "https://broker.acme.internal",
			stderr:    "Broker URL: ",
		},
		{
			name:      "Enter keeps the current value shown in brackets",
			current:   "https://broker.acme.internal",
			stdin:     "\n",
			brokerURL: "https://broker.acme.internal",
			stderr:    "Broker URL [https://broker.acme.internal]: ",
		},
		{
			name:      "a new value replaces the current one",
			current:   "https://broker.acme.internal",
			stdin:     "http://localhost:8080\n",
			brokerURL: "http://localhost:8080",
			stderr:    "Broker URL [https://broker.acme.internal]: ",
		},
		{
			name:      "the current value shows its password hidden",
			current:   "https://ci:segredo@broker.acme.internal",
			stdin:     "\n",
			brokerURL: "https://ci:segredo@broker.acme.internal",
			stderr:    "Broker URL [https://ci:xxxxx@broker.acme.internal]: ",
		},
		{
			name:      "asks again until the URL is valid, without normalizing it",
			stdin:     "broker.acme.com\nftp://x\nhttp://localhost:8080\n",
			brokerURL: "http://localhost:8080",
			stderr: "Broker URL: " +
				`invalid broker URL "broker.acme.com" — use http:// or https:// followed by a host` + "\n" +
				"Broker URL: " +
				`invalid broker URL "ftp://x" — use http:// or https:// followed by a host` + "\n" +
				"Broker URL: ",
		},
		{
			name:      "Enter without a current value asks again",
			stdin:     "\nhttp://localhost:8080\n",
			brokerURL: "http://localhost:8080",
			stderr: "Broker URL: " +
				`invalid broker URL "" — use http:// or https:// followed by a host` + "\n" +
				"Broker URL: ",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stderr bytes.Buffer

			brokerURL, err := components.PromptBrokerURL(strings.NewReader(test.stdin), &stderr, test.current)

			require.NoError(t, err)
			assert.Equal(t, test.brokerURL, brokerURL)
			assert.Equal(t, test.stderr, stderr.String())
		})
	}
}

func TestPromptBrokerURLFailsAtTheEndOfStdin(t *testing.T) {
	for _, stdin := range []string{"", "broker.acme.com\n"} {
		t.Run(stdin, func(t *testing.T) {
			var stderr bytes.Buffer

			_, err := components.PromptBrokerURL(strings.NewReader(stdin), &stderr, "")

			assert.EqualError(t, err, "no broker URL entered")
		})
	}
}
