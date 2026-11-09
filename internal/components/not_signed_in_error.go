package components

type NotSignedInError struct {
	BrokerURL string
}

func (this NotSignedInError) Error() string {
	return "not signed in to " + this.BrokerURL + ` — run "bidirekt login" or set BIDIREKT_BROKER_URL and BIDIREKT_API_KEY`
}

func (this NotSignedInError) UserText() string {
	return this.Error()
}
