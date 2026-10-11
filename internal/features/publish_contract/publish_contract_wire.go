package publish_contract

import (
	"github.com/bidirekt/cli/internal/contractfiles"
	"github.com/bidirekt/cli/internal/reports"
)

const contractParticipantNotFound = "contract participant not found"

type PublishContractRequestBody struct {
	Participant string                           `json:"participant"`
	Version     string                           `json:"version"`
	Contracts   []contractfiles.ContractFragment `json:"contracts"`
}

type PublishContractResponseBody struct {
	Message    string              `json:"message"`
	Violations []reports.Violation `json:"violations"`
}
