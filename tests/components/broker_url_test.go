package components_test

import (
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/stretchr/testify/assert"
)

func TestValidateBrokerURLAcceptsHTTPAndHTTPSWithAHost(t *testing.T) {
	for _, brokerURL := range []string{
		"http://localhost:8080",
		"https://broker.acme.internal",
		"https://ci:segredo@broker.acme.internal",
		"http://127.0.0.1:9000/prefix",
		"http://[::1]:8080",
	} {
		t.Run(brokerURL, func(t *testing.T) {
			assert.NoError(t, components.ValidateBrokerURL(brokerURL, "from --broker-url"))
		})
	}
}

func TestValidateBrokerURLRejectsAnythingElseWithoutNormalizing(t *testing.T) {
	tests := []struct {
		rawURL string
		origin string
		err    string
	}{
		{
			rawURL: "broker.acme.com",
			origin: "from BIDIREKT_BROKER_URL",
			err:    `invalid broker URL "broker.acme.com" (from BIDIREKT_BROKER_URL) — use http:// or https:// followed by a host`,
		},
		{
			rawURL: "ftp://x",
			origin: "from --broker-url",
			err:    `invalid broker URL "ftp://x" (from --broker-url) — use http:// or https:// followed by a host`,
		},
		{
			rawURL: "http://",
			origin: "profile: staging",
			err:    `invalid broker URL "http://" (profile: staging) — use http:// or https:// followed by a host`,
		},
		{
			rawURL: "http://:8080",
			origin: "from --broker-url",
			err:    `invalid broker URL "http://:8080" (from --broker-url) — use http:// or https:// followed by a host`,
		},
		{
			rawURL: "http://localhost:80a",
			origin: "from --broker-url",
			err:    `invalid broker URL "http://localhost:80a" (from --broker-url) — use http:// or https:// followed by a host`,
		},
		{
			rawURL: "https://ci:segredo@",
			origin: "from BIDIREKT_BROKER_URL",
			err:    `invalid broker URL "https://ci:xxxxx@" (from BIDIREKT_BROKER_URL) — use http:// or https:// followed by a host`,
		},
		{
			rawURL: "https://ci:se%gredo@broker.acme.internal",
			origin: "profile: default",
			err:    `invalid broker URL "xxxxx" (profile: default) — use http:// or https:// followed by a host`,
		},
		{
			rawURL: "",
			err:    `invalid broker URL "" — use http:// or https:// followed by a host`,
		},
	}

	for _, test := range tests {
		t.Run(test.rawURL, func(t *testing.T) {
			assert.EqualError(t, components.ValidateBrokerURL(test.rawURL, test.origin), test.err)
		})
	}
}

func TestRedactedURLHidesOnlyThePassword(t *testing.T) {
	tests := []struct {
		rawURL   string
		redacted string
	}{
		{rawURL: "https://ci:segredo@broker.acme.internal", redacted: "https://ci:xxxxx@broker.acme.internal"},
		{rawURL: "https://ci@broker.acme.internal", redacted: "https://ci@broker.acme.internal"},
		{rawURL: "http://localhost:8080", redacted: "http://localhost:8080"},
		{rawURL: "http://", redacted: "http://"},
		{rawURL: "https://ci:se%gredo@broker.acme.internal", redacted: "xxxxx"},
		{rawURL: "http://localhost:80a", redacted: "http://localhost:80a"},
	}

	for _, test := range tests {
		t.Run(test.rawURL, func(t *testing.T) {
			assert.Equal(t, test.redacted, components.RedactedURL(test.rawURL))
		})
	}
}
