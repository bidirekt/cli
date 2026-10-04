package components

// Commands carrying this annotation get their broker URL resolved before they run.
const TalksToBrokerAnnotation = "talks-to-broker"

type Components struct {
	Config     *Config
	HTTPClient *HTTPClient
}

func New() *Components {
	config := NewConfig()
	return &Components{
		HTTPClient: NewHTTPClient(config),
		Config:     config,
	}
}
