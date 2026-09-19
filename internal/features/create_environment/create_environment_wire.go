package create_environment

type CreateEnvironmentRequestBody struct {
	Environment string `json:"environment"`
}

type CreateEnvironmentResponseBody struct {
	Message string `json:"message"`
}
