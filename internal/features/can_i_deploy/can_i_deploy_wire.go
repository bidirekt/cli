package can_i_deploy

import "github.com/bidirekt/cli/internal/reports"

type CanIDeployRequestBody struct {
	Participant string `json:"participant"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
}

type CanIDeployResponseBody struct {
	Message     string                              `json:"message"`
	Deployable  bool                                `json:"deployable"`
	Environment string                              `json:"environment"`
	Results     map[string]reports.CanIDeployResult `json:"results"`
}
