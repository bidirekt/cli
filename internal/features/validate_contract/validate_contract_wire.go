package validate_contract

import (
	"github.com/bidirekt/cli/internal/contractfiles"
	"github.com/bidirekt/cli/internal/reports"
)

type ValidateContractRequestBody struct {
	Participant string                           `json:"participant"`
	Environment string                           `json:"environment"`
	Contracts   []contractfiles.ContractFragment `json:"contracts"`
}

type ValidateContractResponseBody struct {
	Message     string                              `json:"message"`
	Participant string                              `json:"participant"`
	Environment string                              `json:"environment"`
	Deployable  bool                                `json:"deployable"`
	Results     map[string]reports.CanIDeployResult `json:"results"`
	Violations  []reports.Violation                 `json:"violations"`
}
