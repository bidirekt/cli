package components

// Commands carrying this annotation get their profile resolved before they run.
const TalksToBrokerAnnotation = "talks-to-broker"

type Components struct {
	Config     *Config
	HTTPClient *HTTPClient
}

func New() *Components {
	return &Components{
		HTTPClient: NewHTTPClient(),
		Config:     NewConfig(),
	}
}
