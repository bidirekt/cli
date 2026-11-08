package login

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

var errNoProfileEntered = errors.New("no profile entered")

func promptProfile(lines *bufio.Scanner, stderr io.Writer, current string) (string, error) {
	if _, err := fmt.Fprintf(stderr, "Profile [%s]: ", current); err != nil {
		return "", err
	}

	if !lines.Scan() {
		if err := lines.Err(); err != nil {
			return "", fmt.Errorf("read profile: %w", err)
		}
		return "", errNoProfileEntered
	}

	if profile := lines.Text(); profile != "" {
		return profile, nil
	}
	return current, nil
}
