package steps

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	auth "github.com/ONSdigital/dp-authorisation/v2/authorisation"
	authMock "github.com/ONSdigital/dp-authorisation/v2/authorisation/mock"

	vault "github.com/ONSdigital/dp-vault"

	s3client "github.com/ONSdigital/dp-s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"

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
	Server       *dphttp.Server
	isAuthorised bool
}

func (e *External) AuthMiddleware(ctx context.Context, c *config.Config) (auth.Middleware, error) {
	return &authMock.MiddlewareMock{
		HealthCheckFunc: func(ctx context.Context, state *healthcheck.CheckState) error {
			state.Update("OK", "is healthy", 0)
			return nil
		},
		RequireFunc: func(permission string, handlerFunc http.HandlerFunc) http.HandlerFunc {
			if e.isAuthorised {
				return handlerFunc
			}

			return func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			}
		},
	}, nil
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

func (e *External) VaultClient(cfg *config.Config) (content.VaultClient, error) {

	v, err := vault.CreateClient(cfg.VaultToken, cfg.VaultAddress, 5)
	if err != nil {
		fmt.Println(err.Error())
	}

	return v, nil
}

func (e *External) S3Client(cfg *config.Config) (content.S3Client, error) {
	s, err := session.NewSession(&aws.Config{
		Endpoint:         aws.String(localStackHost),
		Region:           aws.String(cfg.AwsRegion),
		S3ForcePathStyle: aws.Bool(true),
		Credentials:      credentials.NewStaticCredentials("test", "test", ""),
	})

	if err != nil {
		fmt.Println("S3 ERROR: " + err.Error())
	}

	return s3client.NewClientWithSession(cfg.BucketName, s), nil
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
