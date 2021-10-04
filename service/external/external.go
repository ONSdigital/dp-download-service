package external

import (
	"context"
	"fmt"

	"github.com/ONSdigital/dp-api-clients-go/v2/dataset"
	"github.com/ONSdigital/dp-api-clients-go/v2/filter"
	"github.com/ONSdigital/dp-api-clients-go/v2/image"
	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/content"
	"github.com/ONSdigital/dp-download-service/downloads"
	"github.com/ONSdigital/dp-download-service/mongo"
	"github.com/ONSdigital/dp-download-service/service"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	s3client "github.com/ONSdigital/dp-s3"
	vault "github.com/ONSdigital/dp-vault"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
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

// S3Client obtains a new S3 client, or a local storage client if a non-empty LocalObjectStore is provided
func (*External) S3Client(cfg *config.Config) (content.S3Client, error) {
	if cfg.LocalObjectStore != "" {
		s3Config := &aws.Config{
			Credentials:      credentials.NewStaticCredentials(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
			Endpoint:         aws.String(cfg.LocalObjectStore),
			Region:           aws.String(cfg.AwsRegion),
			DisableSSL:       aws.Bool(true),
			S3ForcePathStyle: aws.Bool(true),
		}

		s, err := session.NewSession(s3Config)
		if err != nil {
			return nil, fmt.Errorf("could not create the local-object-store s3 client: %w", err)
		}
		return s3client.NewClientWithSession(cfg.BucketName, s), nil
	}

	s3, err := s3client.NewClient(cfg.AwsRegion, cfg.BucketName)
	if err != nil {
		return nil, fmt.Errorf("could not create the s3 client: %w", err)
	}

	return s3, nil
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