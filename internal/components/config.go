package components

import "os"

type Config struct {
	BrokerURL string
}

func NewConfig() *Config {
	brokerURL := os.Getenv("BIDIREKT_BROKER_URL")

	if brokerURL == "" {
		brokerURL = "http://localhost:8080"
	}

	return &Config{
		BrokerURL: brokerURL,
	}
}
