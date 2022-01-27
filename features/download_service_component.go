package features

import (
	"bytes"
	"context"
	"fmt"
	componenttest "github.com/ONSdigital/dp-component-test"
	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/content"
	contentMocks "github.com/ONSdigital/dp-download-service/content/mocks"
	"github.com/ONSdigital/dp-download-service/downloads"
	mocks "github.com/ONSdigital/dp-download-service/downloads/mocks"
	"github.com/ONSdigital/dp-download-service/service"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	dphttp "github.com/ONSdigital/dp-net/v2/http"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"net/http"
	"os"
	"testing"
	"time"
)

type DownloadServiceComponent struct {
	DpHttpServer *dphttp.Server
	svc        *service.Download
	ApiFeature *componenttest.APIFeature
	errChan    chan error
}

type External struct {
	Server *dphttp.Server
}

func (e *External) FilterClient(s string) downloads.FilterClient {
	t := &testing.T{}
	c := gomock.NewController(t)
	m := mocks.NewMockFilterClient(c)
	m.EXPECT().Checker(gomock.Any(), gomock.Any()).Return(nil)
	return m
}

func (e *External) ImageClient(s string) downloads.ImageClient {
	t := &testing.T{}
	c := gomock.NewController(t)
	m := mocks.NewMockImageClient(c)
	m.EXPECT().Checker(gomock.Any(), gomock.Any()).Return(nil)
	return m
}

func (e *External) VaultClient(cfg *config.Config) (content.VaultClient, error) {
	t := &testing.T{}
	c := gomock.NewController(t)
	m := contentMocks.NewMockVaultClient(c)
	m.EXPECT().Checker(gomock.Any(), gomock.Any()).Return(nil)
	return m, nil
}

func (e *External) S3Client(cfg *config.Config) (content.S3Client, error) {
	t := &testing.T{}
	c := gomock.NewController(t)
	m := contentMocks.NewMockS3Client(c)
	m.EXPECT().Checker(gomock.Any(), gomock.Any()).Return(nil)
	return m, nil
}

func (e *External) HealthCheck(c *config.Config, s string, s2 string, s3 string) (service.HealthChecker, error) {
	hc := healthcheck.New(healthcheck.VersionInfo{}, time.Second, time.Second)
	return &hc, nil
}

func (e *External) DatasetClient(datasetAPIURL string) downloads.DatasetClient {
	t := &testing.T{}
	c := gomock.NewController(t)
	m := mocks.NewMockDatasetClient(c)
	m.EXPECT().Checker(gomock.Any(), gomock.Any()).Return(nil)
	return m
}

func (e *External) HttpServer(cfg *config.Config, r http.Handler) service.HTTPServer {
	e.Server.Server.Addr = cfg.BindAddr
	e.Server.Server.Handler = r

	return e.Server
}

func NewDownloadServiceComponent(fake_auth_url string) *DownloadServiceComponent {
	buf := bytes.NewBufferString("")
	log.SetDestination(buf, buf)
	os.Setenv("ZEBEDEE_URL", fake_auth_url)

	d := &DownloadServiceComponent{
		DpHttpServer: dphttp.NewServer("", http.NewServeMux()),
		errChan:    make(chan error),
	}

	fmt.Println("handler created in new", d.DpHttpServer.Server.Handler)

	//d.HttpServer.Handler = handler

	os.Setenv("DATABASE_NAME", "testing")

	log.Namespace = "dp-download-service"

	ctx := context.Background()

	cfg, err := config.Get()
	assert.NoError(&componenttest.ErrorFeature{}, err, "error getting config")

	log.Info(ctx, "config on startup", log.Data{"config": cfg})

	d.svc, err = service.New(ctx, "1", "1", "1", cfg, &External{Server: d.DpHttpServer})

	if err != nil {
		log.Fatal(ctx, "could not set up Download service", err)
		os.Exit(1)
	}

	return d
}

func (d *DownloadServiceComponent) Initialiser() (http.Handler, error) {
	d.svc.Run(context.Background())

	return d.DpHttpServer.Handler, nil
}
