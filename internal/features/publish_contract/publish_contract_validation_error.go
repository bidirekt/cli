package publish_contract

type ValidationFailedError struct {
	Message    string
	Violations []Violation
}

func (this *ValidationFailedError) Error() string {
	return this.Message
}
