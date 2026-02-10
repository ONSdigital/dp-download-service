package config

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/ONSdigital/dp-authorisation/v2/authorisation"
	"github.com/ONSdigital/log.go/v2/log"

	"github.com/kelseyhightower/envconfig"
)

// Config represents the configuration required for the dp-download-service
type Config struct {
	BindAddr                   string        `envconfig:"BIND_ADDR"`
	AwsRegion                  string        `envconfig:"AWS_REGION"`
	BucketName                 string        `envconfig:"BUCKET_NAME"`
	DatasetAPIURL              string        `envconfig:"DATASET_API_URL"`
	DownloadServiceToken       string        `envconfig:"DOWNLOAD_SERVICE_TOKEN"     json:"-"`
	DatasetAuthToken           string        `envconfig:"DATASET_AUTH_TOKEN"         json:"-"`
	FilesAPIURL                string        `envconfig:"FILES_API_URL"`
	FilterAPIURL               string        `envconfig:"FILTER_API_URL"`
	ImageAPIURL                string        `envconfig:"IMAGE_API_URL"`
	GracefulShutdownTimeout    time.Duration `envconfig:"GRACEFUL_SHUTDOWN_TIMEOUT"  json:"-"`
	HealthCheckInterval        time.Duration `envconfig:"HEALTHCHECK_INTERVAL"`
	HealthCheckCriticalTimeout time.Duration `envconfig:"HEALTHCHECK_CRITICAL_TIMEOUT"`
	OTBatchTimeout             time.Duration `encconfig:"OTEL_BATCH_TIMEOUT"`
	OTExporterOTLPEndpoint     string        `envconfig:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	OTServiceName              string        `envconfig:"OTEL_SERVICE_NAME"`
	OtelEnabled                bool          `envconfig:"OTEL_ENABLED"`
	ServiceAuthToken           string        `envconfig:"SERVICE_AUTH_TOKEN"         json:"-"`
	SecretKey                  string        `envconfig:"SECRET_KEY"                 json:"-"`
	ZebedeeURL                 string        `envconfig:"ZEBEDEE_URL"`
	LocalObjectStore           string        `envconfig:"LOCAL_OBJECT_STORE"` // TODO remove replacing minio with localstack in component tests
	MinioAccessKey             string        `envconfig:"MINIO_ACCESS_KEY"`   // TODO remove replacing minio with localstack in component tests
	MinioSecretKey             string        `envconfig:"MINIO_SECRET_KEY"`   // TODO remove replacing minio with localstack in component tests
	IsPublishing               bool          `envconfig:"IS_PUBLISHING"`
	PublicBucketURL            ConfigUrl     `envconfig:"PUBLIC_BUCKET_URL"`
	MaxConcurrentHandlers      int           `envconfig:"MAX_CONCURRENT_HANDLERS"`
	AuthorisationConfig        *authorisation.Config
}

var cfg *Config

type ConfigUrl struct {
	url.URL
}

func (c *ConfigUrl) Decode(value string) error {
	parsedURL, err := url.Parse(value)
	if err != nil {
		log.Error(context.Background(), fmt.Sprintf("Bad public bucket url: %s", value), err)
		return err
	}

	c.URL = *parsedURL

	return nil
}

// Get retrieves the config from the environment for the dp-download-service
func Get() (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	cfg = &Config{
		BindAddr:                   "localhost:23600",
		AwsRegion:                  "eu-west-2",
		BucketName:                 "csv-exported",
		DatasetAPIURL:              "http://localhost:22000",
		FilesAPIURL:                "http://localhost:26900",
		FilterAPIURL:               "http://localhost:22100",
		ImageAPIURL:                "http://localhost:24700",
		DatasetAuthToken:           "FD0108EA-825D-411C-9B1D-41EF7727F465",
		DownloadServiceToken:       "QB0108EZ-825D-412C-9B1D-41EF7747F462",
		GracefulShutdownTimeout:    5 * time.Second,
		HealthCheckInterval:        30 * time.Second,
		HealthCheckCriticalTimeout: 90 * time.Second,
		OTBatchTimeout:             5 * time.Second,
		OTExporterOTLPEndpoint:     "localhost:4317",
		OTServiceName:              "dp-download-service",
		OtelEnabled:                false,
		ServiceAuthToken:           "c60198e9-1864-4b68-ad0b-1e858e5b46a4",
		ZebedeeURL:                 "http://localhost:8082",
		LocalObjectStore:           "",
		MinioAccessKey:             "",
		MinioSecretKey:             "",
		IsPublishing:               true,
		PublicBucketURL:            ConfigUrl{},
		MaxConcurrentHandlers:      0, // unlimited
		AuthorisationConfig:        authorisation.NewDefaultConfig(),
	}

	if err := envconfig.Process("", cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}
