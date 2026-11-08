package login

type EnvironmentsResponseBody struct {
	Message      string   `json:"message"`
	Environments []string `json:"environments"`
}
