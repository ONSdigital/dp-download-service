package external

import (
	"context"
	"fmt"
	"net/http"

	dphttp "github.com/ONSdigital/dp-net/v2/http"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/ONSdigital/dp-api-clients-go/v2/dataset"
	"github.com/ONSdigital/dp-api-clients-go/v2/files"
	"github.com/ONSdigital/dp-api-clients-go/v2/filter"
	"github.com/ONSdigital/dp-api-clients-go/v2/image"

	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/content"
	"github.com/ONSdigital/dp-download-service/downloads"
	"github.com/ONSdigital/dp-download-service/service"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	s3client "github.com/ONSdigital/dp-s3/v3"
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

func (*External) FilesClient(cfg *config.Config) downloads.FilesClient {
	return files.NewAPIClient(cfg.FilesApiURL, cfg.ServiceAuthToken)
}

// S3Client obtains a new S3 client, or a local storage client if a non-empty LocalObjectStore is provided
func (*External) S3Client(ctx context.Context, cfg *config.Config) (content.S3Client, error) {
	if cfg.LocalObjectStore != "" {
		awsConfig, err := awsConfig.LoadDefaultConfig(ctx,
			awsConfig.WithRegion(cfg.AwsRegion),
			awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.MinioAccessKey, cfg.MinioSecretKey, "")),
		)
		if err != nil {
			return nil, fmt.Errorf("could not create aws config: %w", err)
		}

		s3client := s3client.NewClientWithConfig(cfg.BucketName, awsConfig, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.LocalObjectStore)
			o.UsePathStyle = true
		})

		return s3client, nil
	}

	s3client, err := s3client.NewClient(ctx, cfg.AwsRegion, cfg.BucketName)
	if err != nil {
		return nil, fmt.Errorf("could not create s3 client: %w", err)
	}

	return s3client, nil
}

func (*External) HealthCheck(cfg *config.Config, buildTime, gitCommit, version string) (service.HealthChecker, error) {
	versionInfo, err := healthcheck.NewVersionInfo(buildTime, gitCommit, version)
	if err != nil {
		return nil, err
	}
	hc := healthcheck.New(versionInfo, cfg.HealthCheckCriticalTimeout, cfg.HealthCheckInterval)
	return &hc, nil
}

func (*External) HttpServer(cfg *config.Config, r http.Handler) service.HTTPServer {
	s := dphttp.NewServer(cfg.BindAddr, r)
	s.HandleOSSignals = false

	return s
}
