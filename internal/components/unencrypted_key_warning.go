package components

import (
	"fmt"
	"io"
	"net/url"
	"strings"
)

func WarnIfKeyTravelsUnencrypted(stderr io.Writer, brokerURL, apiKey string) error {
	if apiKey == "" {
		return nil
	}

	parsed, err := url.Parse(brokerURL)
	if err != nil || parsed.Scheme != "http" {
		return nil
	}
	if host := parsed.Hostname(); strings.EqualFold(host, "localhost") || host == "127.0.0.1" {
		return nil
	}

	_, err = fmt.Fprintf(stderr, "warning: the API key travels unencrypted to %s\n", RedactedURL(brokerURL))
	return err
}
