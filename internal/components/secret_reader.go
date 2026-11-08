package components

import (
	"errors"
	"fmt"
	"os"
	"os/signal"

	"golang.org/x/term"
)

const interruptedExitCode = 130

var errNotATerminal = errors.New("read secret: stdin is not a terminal")

func ReadSecret(stream any) (string, error) {
	file, ok := stream.(*os.File)
	if !ok {
		return "", errNotATerminal
	}
	fd := int(file.Fd())

	state, err := term.GetState(fd)
	if err != nil {
		return "", errNotATerminal
	}

	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	finished := make(chan struct{})
	defer close(finished)
	defer signal.Stop(interrupts)
	go func() {
		select {
		case <-interrupts:
			_ = term.Restore(fd, state)
			os.Exit(interruptedExitCode)
		case <-finished:
		}
	}()

	secret, err := term.ReadPassword(fd)
	if err != nil {
		return "", fmt.Errorf("read secret: %w", err)
	}

	return string(secret), nil
}
