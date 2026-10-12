package reports

import "fmt"

func NewParticipantNotFoundError(participant string) error {
	return fmt.Errorf("participant %q not found → create it with: bidirekt create-participant %s", participant, participant)
}
