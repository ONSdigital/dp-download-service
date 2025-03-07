package steps

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	s3client "github.com/ONSdigital/dp-s3/v3"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/content"
	"github.com/ONSdigital/dp-download-service/downloads"
	"github.com/ONSdigital/dp-download-service/downloads/mocks"
	"github.com/ONSdigital/dp-download-service/service"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	dphttp "github.com/ONSdigital/dp-net/v2/http"
	"github.com/golang/mock/gomock"
)

type External struct {
	Server *dphttp.Server
}

func (e *External) FilterClient(s string) downloads.FilterClient {
	t := &testing.T{}
	c := gomock.NewController(t)
	m := mocks.NewMockFilterClient(c)
	m.EXPECT().Checker(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, check *healthcheck.CheckState) error {
		check.Update("OK", "MsgHealthy", 0)
		return nil
	})
	return m
}

func (e *External) ImageClient(s string) downloads.ImageClient {
	t := &testing.T{}
	c := gomock.NewController(t)
	m := mocks.NewMockImageClient(c)
	m.EXPECT().Checker(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, check *healthcheck.CheckState) error {
		check.Update("OK", "MsgHealthy", 0)
		return nil
	})
	return m
}

func (e *External) FilesClient(cfg *config.Config) downloads.FilesClient {
	t := &testing.T{}
	c := gomock.NewController(t)
	m := mocks.NewMockFilesClient(c)
	m.EXPECT().Checker(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, check *healthcheck.CheckState) error {
		check.Update("OK", "MsgHealthy", 0)
		return nil
	})
	return m
}

func (e *External) S3Client(cfg *config.Config) (content.S3Client, error) {
	ctx := context.Background()

	awsConfig, err := awsConfig.LoadDefaultConfig(ctx,
		awsConfig.WithRegion(cfg.AwsRegion),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	if err != nil {
		return nil, fmt.Errorf("could not create aws config: %w", err)
	}

	s3client := s3client.NewClientWithConfig(cfg.BucketName, awsConfig, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(localStackHost)
		o.UsePathStyle = true
	})

	return s3client, nil
}

func (e *External) HealthCheck(c *config.Config, s string, s2 string, s3 string) (service.HealthChecker, error) {
	hc := healthcheck.New(healthcheck.VersionInfo{}, time.Second, time.Second)
	return &hc, nil
}

func (e *External) DatasetClient(datasetAPIURL string) downloads.DatasetClient {
	t := &testing.T{}
	c := gomock.NewController(t)
	m := mocks.NewMockDatasetClient(c)
	m.EXPECT().Checker(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, check *healthcheck.CheckState) error {
		check.Update("OK", "MsgHealthy", 0)
		return nil
	})
	return m
}

func (e *External) HttpServer(cfg *config.Config, r http.Handler) service.HTTPServer {
	e.Server.Server.Addr = cfg.BindAddr
	e.Server.Server.Handler = r

	return e.Server
}
