package components

type UpgradeRequiredError struct {
	Message string `json:"message"`
}

func (this UpgradeRequiredError) Error() string {
	return this.Message
}

func (this UpgradeRequiredError) UserText() string {
	return this.Error()
}
