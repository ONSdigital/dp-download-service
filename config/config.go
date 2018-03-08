package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config represents the configuration required for the dp-download-service
type Config struct {
	BindAddr                string        `envconfig:"BIND_ADDR"`
	BucketName              string        `envconfig:"BUCKET_NAME"`
	EncryptionDisabled      bool          `envconfig:"ENCRYPTION_DISABLED"`
	DatasetAPIURL           string        `envconfig:"DATASET_API_URL"`
	DatasetAuthToken        string        `envconfig:"DATASET_AUTH_TOKEN"     json:"-"`
	GracefulShutdownTimeout time.Duration `envconfig:"GRACEFUL_SHUTDOWN_TIMEOUT"`
	HealthCheckInterval     time.Duration `envconfig:"HEALTHCHECK_INTERVAL"`
	SecretKey               string        `envconfig:"SECRET_KEY"             json:"-"`
	VaultToken              string        `envconfig:"VAULT_TOKEN"            json:"-"`
	VaultAddress            string        `envconfig:"VAULT_ADDR"`
	VaultPath               string        `envconfig:"VAULT_PATH"`
}

var cfg *Config

// Get retrieves the config from the environment for the dp-download-service
func Get() (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	cfg = &Config{
		BindAddr:                ":28000",
		EncryptionDisabled:      true,
		DatasetAPIURL:           "http://localhost:22000",
		DatasetAuthToken:        "FD0108EA-825D-411C-9B1D-41EF7727F46",
		GracefulShutdownTimeout: 5 * time.Second,
		HealthCheckInterval:     1 * time.Minute,
		SecretKey:               "AL0108EA-825D-411C-9B1D-41EF7727F46",
		VaultAddress:            "http://localhost:8200",
	}

	return cfg, envconfig.Process("", cfg)
}
