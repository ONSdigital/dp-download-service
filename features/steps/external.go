package steps

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/ONSdigital/dp-api-clients-go/v2/files"
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
	dphttp "github.com/ONSdigital/dp-net/v3/http"
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

	m.EXPECT().Checker(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, check *healthcheck.CheckState) error {
			check.Update("OK", "MsgHealthy", 0)
			return nil
		})

	m.EXPECT().GetFile(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, path, authToken string) (files.FileMetaData, error) {
			switch path {
			case "data/published.csv":
				return files.FileMetaData{State: "PUBLISHED", Path: path, Type: "text/csv", SizeInBytes: 29}, nil
			case "data/unpublished.csv":
				return files.FileMetaData{State: "UPLOADED", Path: path, Type: "text/csv", SizeInBytes: 29}, nil
			case "data/weird&chars#published.csv":
				return files.FileMetaData{State: "PUBLISHED", Path: path, Type: "text/csv", SizeInBytes: 29}, nil
			case "data/weird&chars#unpublished.csv":
				return files.FileMetaData{State: "UPLOADED", Path: path, Type: "text/csv", SizeInBytes: 29}, nil
			case "data/return301.csv":
				return files.FileMetaData{State: "MOVED", Path: path, Type: "text/csv", SizeInBytes: 29}, nil
			case "data/test.csv":
				return files.FileMetaData{State: "PUBLISHED", Path: path, Type: "text/csv", SizeInBytes: 10}, nil
			case "data/uploaded.csv":
				return files.FileMetaData{State: "UPLOADED", Path: path, Type: "text/csv", SizeInBytes: 15}, nil
			case "data/missing.csv":
				return files.FileMetaData{}, fmt.Errorf("file not registered")
			default:
				return files.FileMetaData{}, fmt.Errorf("unknown mock path")
			}
		})

	m.EXPECT().CreateFileEvent(gomock.Any(), gomock.Any()).AnyTimes().Return(nil, nil)

	return m
}

func (e *External) S3Client(ctx context.Context, cfg *config.Config) (content.S3Client, error) {
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
