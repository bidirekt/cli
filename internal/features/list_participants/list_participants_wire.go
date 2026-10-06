package list_participants

type ListParticipantsResponseBody struct {
	Message      string   `json:"message"`
	Participants []string `json:"participants"`
}
