package config

import (
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config represents the configuration required for the dp-download-service
type Config struct {
	BindAddr                string        `envconfig:"BIND_ADDR"`
	BucketName              string        `envconfig:"BUCKET_NAME"`
	DatasetAPIURL           string        `envconfig:"DATASET_API_URL"`
	DownloadServiceToken    string        `envconfig:"DOWNLOAD_SERVICE_TOKEN"     json:"-"`
	DatasetAuthToken        string        `envconfig:"DATASET_AUTH_TOKEN"         json:"-"`
	FilterAPIURL            string        `envconfig:"FILTER_API_URL"`
	GracefulShutdownTimeout time.Duration `envconfig:"GRACEFUL_SHUTDOWN_TIMEOUT"  json:"-"`
	HealthCheckInterval     time.Duration `envconfig:"HEALTHCHECK_INTERVAL"`
	HealthCheckRecovery     time.Duration `envconfig:"HEALTHCHECK_RECOVERY_INTERVAL"`
	SecretKey               string        `envconfig:"SECRET_KEY"                 json:"-"`
	VaultToken              string        `envconfig:"VAULT_TOKEN"                json:"-"`
	VaultAddress            string        `envconfig:"VAULT_ADDR"`
	VaultPath               string        `envconfig:"VAULT_PATH"`
	ServiceAuthToken        string        `envconfig:"SERVICE_AUTH_TOKEN"         json:"-"`
	ZebedeeURL              string        `envconfig:"ZEBEDEE_URL"`
	IsPublishing            bool          `envconfig:"IS_PUBLISHING"`
}

var cfg *Config

// Get retrieves the config from the environment for the dp-download-service
func Get() (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	cfg = &Config{
		BindAddr:                ":23600",
		BucketName:              "csv-exported",
		DatasetAPIURL:           "http://localhost:22000",
		FilterAPIURL:            "http://localhost:22100",
		DatasetAuthToken:        "FD0108EA-825D-411C-9B1D-41EF7727F465",
		DownloadServiceToken:    "QB0108EZ-825D-412C-9B1D-41EF7747F462",
		GracefulShutdownTimeout: 5 * time.Second,
		HealthCheckInterval:     1 * time.Minute,
		HealthCheckRecovery:     5 * time.Second,
		VaultAddress:            "http://localhost:8200",
		VaultToken:              "",
		VaultPath:               "secret/shared/psk",
		ServiceAuthToken:        "c60198e9-1864-4b68-ad0b-1e858e5b46a4",
		ZebedeeURL:              "http://localhost:8082",
		IsPublishing:            true,
	}

	if err := envconfig.Process("", cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}
