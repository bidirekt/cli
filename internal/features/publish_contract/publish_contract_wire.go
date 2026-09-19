package publish_contract

type ContractFragment struct {
	Source  string `json:"source"`
	Content string `json:"content"`
}

type PublishContractRequestBody struct {
	Participant string             `json:"participant"`
	Version     string             `json:"version"`
	Contracts   []ContractFragment `json:"contracts"`
}

type PublishContractResponseBody struct {
	Message    string      `json:"message"`
	Violations []Violation `json:"violations"`
}

type Violation struct {
	Code    string            `json:"code"`
	Path    string            `json:"path"`
	Source  string            `json:"source"`
	Details map[string]string `json:"details"`
}
