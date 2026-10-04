package components

import (
	"fmt"
	"net/url"
	"strings"
)

func ValidateBrokerURL(rawURL, origin string) error {
	parsed, err := url.Parse(rawURL)
	if err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Hostname() != "" {
		return nil
	}

	if origin == "" {
		return fmt.Errorf("invalid broker URL %q — use http:// or https:// followed by a host", RedactedURL(rawURL))
	}
	return fmt.Errorf("invalid broker URL %q (%s) — use http:// or https:// followed by a host", RedactedURL(rawURL), origin)
}

// RedactedURL returns rawURL untouched unless it carries a password, because url.URL.String
// rewrites what it serializes ("http://" becomes "http:"). A value url.Parse rejects cannot be
// split into its parts, so it is masked whole whenever it may carry credentials.
func RedactedURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		if strings.Contains(rawURL, "@") {
			return "xxxxx"
		}
		return rawURL
	}

	if _, hasPassword := parsed.User.Password(); hasPassword {
		return parsed.Redacted()
	}
	return rawURL
}
