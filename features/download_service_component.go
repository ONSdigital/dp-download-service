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
	"github.com/ONSdigital/dp-download-service/downloads/mocks"
	"github.com/ONSdigital/dp-download-service/service"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	dphttp "github.com/ONSdigital/dp-net/v2/http"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/stretchr/testify/assert"
	"net/http"
	"os"
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
	return &mocks.MockFilterClient{}
}

func (e *External) ImageClient(s string) downloads.ImageClient {
	return &mocks.MockImageClient{}
}

func (e *External) VaultClient(c *config.Config) (content.VaultClient, error) {
	return &contentMocks.MockVaultClient{}, nil
}

func (e *External) S3Client(c *config.Config) (content.S3Client, error) {
	return &contentMocks.MockS3Client{}, nil
}

func (e *External) MongoClient(ctx context.Context, c *config.Config) (service.MongoClient, error) {
	panic("implement me")
}

func (e *External) HealthCheck(c *config.Config, s string, s2 string, s3 string) (service.HealthChecker, error) {
	hc := healthcheck.New(healthcheck.VersionInfo{}, time.Second, time.Second)
	return &hc, nil
}

func (e *External) DatasetClient(datasetAPIURL string) downloads.DatasetClient {
	return &mocks.MockDatasetClient{}
}

func (e *External) HttpServer(cfg *config.Config, r http.Handler) service.HTTPServer {
	e.Server.Server.Addr = cfg.BindAddr
	e.Server.Server.Handler = r

	return e.Server
	//e.server.Addr = cfg.BindAddr
	//e.server.Handler = r
	//
	//return e.server
}

func NewDownloadServiceComponent(mongoUrl string, fake_auth_url string) *DownloadServiceComponent {
	buf := bytes.NewBufferString("")
	log.SetDestination(buf, buf)
	os.Setenv("ZEBEDEE_URL", fake_auth_url)

	d := &DownloadServiceComponent{
		DpHttpServer: dphttp.NewServer("", http.NewServeMux()),
		errChan:    make(chan error),
	}

	fmt.Println("handler created in new", d.DpHttpServer.Server.Handler)

	//d.HttpServer.Handler = handler

	os.Setenv("MONGO_URL", mongoUrl)
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

	//svc.Run(ctx)

	return d
}

func (d *DownloadServiceComponent) Initialiser() (http.Handler, error) {
	d.svc.Run(context.Background())

	//fmt.Println("handler: ", d.HttpServer.Handler)
	//d.ServiceRunning = true
	return d.DpHttpServer.Handler, nil
}
