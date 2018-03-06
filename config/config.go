package config

import (
	"github.com/kelseyhightower/envconfig"
)

// Config represents the configuration required for the dp-download-service
type Config struct {
	BindAddr           string `envconfig:"BIND_ADDR"`
	EncryptionDisabled bool   `envconfig:"ENCRYPTION_DISABLED"`
	PrivateKey         string `envconfig:"RSA_PRIVATE_KEY"`
	DatasetAPIURL      string `envconfig:"DATASET_API_URL"`
	DatasetAuthToken   string `envconfig:"DATASET_AUTH_TOKEN"`
	SecretKey          string `envconfig:"SECRET_KEY"`
	VaultToken         string `envconfig:"VAULT_TOKEN"`
	VaultAddress       string `envconfig:"VAULT_ADDR"`
}

var cfg *Config

// Get retrieves the config from the environment for the dp-download-service
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
		VaultAddress:       "http://localhost:8200",
	}

	return cfg, envconfig.Process("", cfg)
}
