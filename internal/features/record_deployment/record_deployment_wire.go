package record_deployment

const deploymentRolledBack = "deployment rolled back"

type RecordDeploymentRequestBody struct {
	Participant string `json:"participant"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
}

type RecordDeploymentResponseBody struct {
	Message string `json:"message"`
}
