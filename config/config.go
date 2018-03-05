package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	BindAddr           string `envconfig:"BIND_ADDR"`
	EncryptionDisabled bool   `envconfig:"ENCRYPTION_DISABLED"`
	PrivateKey         string `envconfig:"RSA_PRIVATE_KEY"`
	DatasetAPIURL      string `envconfig:"DATASET_API_URL"`
	DatasetAuthToken   string `envconfig:"DATASET_AUTH_TOKEN"`
	SecretKey          string `envconfig:"SECRET_KEY" `
}

var cfg *Config

func Get() (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	cfg = &Config{
		BindAddr:           ":28000",
		EncryptionDisabled: true,
		PrivateKey:         "",
		DatasetAPIURL:      "http://localhost:22000",
		DatasetAuthToken:   "FD0108EA-825D-411C-9B1D-41EF7727F46",
		SecretKey:          "AL0108EA-825D-411C-9B1D-41EF7727F46",
	}

	return cfg, envconfig.Process("", cfg)
}
