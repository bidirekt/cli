package components

import "os"

type Config struct {
	BrokerURL string
	APIKey    string
	Profile   string
}

func NewConfig() *Config {
	return &Config{
		BrokerURL: os.Getenv("BIDIREKT_BROKER_URL"),
		APIKey:    os.Getenv("BIDIREKT_API_KEY"),
		Profile:   os.Getenv("BIDIREKT_PROFILE"),
	}
}
