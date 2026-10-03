package can_i_deploy

type CanIDeployRequestBody struct {
	Participant string `json:"participant"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
}

type CanIDeployResponseBody struct {
	Message     string                      `json:"message"`
	Deployable  bool                        `json:"deployable"`
	Environment string                      `json:"environment"`
	Results     map[string]CanIDeployResult `json:"results"`
}

// Endpoints nest as endpoint -> method -> interaction -> breaks, where interaction is
// "request" or a response status code ("200").
type CanIDeployResult struct {
	Deployable         bool                                             `json:"deployable"`
	ParticipantVersion *string                                          `json:"participantVersion"`
	Endpoints          map[string]map[string]map[string][]ContractBreak `json:"endpoints"`
}

type ContractBreak struct {
	Reason  string            `json:"reason"`
	Role    string            `json:"role"`
	Details map[string]string `json:"details"`
}
