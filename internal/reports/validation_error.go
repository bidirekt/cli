package reports

type ValidationFailedError struct {
	Message    string
	Violations []Violation
}

func (this *ValidationFailedError) Error() string {
	return this.Message
}
