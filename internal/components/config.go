package components

import "os"

type Config struct {
	BrokerURL string
	Profile   string
}

func NewConfig() *Config {
	return &Config{
		BrokerURL: os.Getenv("BIDIREKT_BROKER_URL"),
		Profile:   os.Getenv("BIDIREKT_PROFILE"),
	}
}
