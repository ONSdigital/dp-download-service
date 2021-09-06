package external

import (
	"context"

	"github.com/ONSdigital/dp-api-clients-go/dataset"
	"github.com/ONSdigital/dp-api-clients-go/filter"
	"github.com/ONSdigital/dp-api-clients-go/image"
	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/content"
	"github.com/ONSdigital/dp-download-service/downloads"
	"github.com/ONSdigital/dp-download-service/mongo"
	"github.com/ONSdigital/dp-download-service/service"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	s3client "github.com/ONSdigital/dp-s3"
	vault "github.com/ONSdigital/dp-vault"
)

// External implements the service.Dependencies interface for actual external services.
type External struct{}

var _ service.Dependencies = &External{}

func (*External) DatasetClient(datasetAPIURL string) downloads.DatasetClient {
	return dataset.NewAPIClient(datasetAPIURL)
}

func (*External) FilterClient(filterAPIURL string) downloads.FilterClient {
	return filter.New(filterAPIURL)
}

func (*External) ImageClient(imageAPIURL string) downloads.ImageClient {
	return image.NewAPIClient(imageAPIURL)
}

func (*External) VaultClient(cfg *config.Config) (content.VaultClient, error) {
	return vault.CreateClient(cfg.VaultToken, cfg.VaultAddress, 3)
}

func (*External) S3Client(cfg *config.Config) (content.S3Client, error) {
	return s3client.NewClient(cfg.AwsRegion, cfg.BucketName, !cfg.EncryptionDisabled)
}

func (ext *External) MongoClient(ctx context.Context, cfg *config.Config) (service.MongoClient, error) {
	return mongo.New(ctx, cfg)
}

func (*External) HealthCheck(cfg *config.Config, buildTime, gitCommit, version string) (service.HealthChecker, error) {
	versionInfo, err := healthcheck.NewVersionInfo(buildTime, gitCommit, version)
	if err != nil {
		return nil, err
	}
	hc := healthcheck.New(versionInfo, cfg.HealthCheckCriticalTimeout, cfg.HealthCheckInterval)
	return &hc, nil
}
