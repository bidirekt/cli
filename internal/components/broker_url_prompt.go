package components

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

func PromptBrokerURL(stdin io.Reader, stderr io.Writer, current string) (string, error) {
	label := "Broker URL: "
	if current != "" {
		label = fmt.Sprintf("Broker URL [%s]: ", RedactedURL(current))
	}

	lines := bufio.NewScanner(stdin)
	for {
		if _, err := fmt.Fprint(stderr, label); err != nil {
			return "", err
		}

		if !lines.Scan() {
			if err := lines.Err(); err != nil {
				return "", fmt.Errorf("read broker URL: %w", err)
			}
			return "", errors.New("no broker URL entered")
		}

		brokerURL := lines.Text()
		if brokerURL == "" {
			brokerURL = current
		}

		if err := ValidateBrokerURL(brokerURL, ""); err != nil {
			if _, err := fmt.Fprintln(stderr, err); err != nil {
				return "", err
			}
			continue
		}

		return brokerURL, nil
	}
}
