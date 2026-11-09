package components

import "strings"

type ForbiddenError struct {
	BrokerURL string
}

func (this ForbiddenError) Error() string {
	return "the API key can't do this — check its permission at " + strings.TrimSuffix(this.BrokerURL, "/") + "/profile"
}

func (this ForbiddenError) UserText() string {
	return this.Error()
}
