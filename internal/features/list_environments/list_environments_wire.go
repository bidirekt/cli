package list_environments

type ListEnvironmentsResponseBody struct {
	Message      string   `json:"message"`
	Environments []string `json:"environments"`
}
