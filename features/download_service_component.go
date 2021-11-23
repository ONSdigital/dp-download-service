package features

import (
	"context"
	"fmt"
	componenttest "github.com/ONSdigital/dp-component-test"
	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/service"
	"github.com/ONSdigital/dp-download-service/service/external"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/stretchr/testify/assert"
	"net/http"
	"os"
)

type DownloadServiceComponent struct {
	Handler http.Handler
}

//
//type External struct{}
//
//func (e *External) FilterClient(s string) downloads.FilterClient {
//	return &mocks.MockFilterClient{}
//}
//
//func (e *External) ImageClient(s string) downloads.ImageClient {
//	return &mocks.MockImageClient{}
//}
//
//func (e *External) VaultClient(c *config.Config) (content.VaultClient, error) {
//	return &contentMocks.MockVaultClient{}, nil
//}
//
//func (e *External) S3Client(c *config.Config) (content.S3Client, error) {
//	return &contentMocks.MockS3Client{}, nil
//}
//
//func (e *External) MongoClient(ctx context.Context, c *config.Config) (service.MongoClient, error) {
//	panic("implement me")
//}
//
//func (e *External) HealthCheck(c *config.Config, s string, s2 string, s3 string) (service.HealthChecker, error) {
//	hc := healthcheck.New(healthcheck.VersionInfo{}, time.Second, time.Second)
//	return &hc, nil
//}
//
//func (e *External) DatasetClient(datasetAPIURL string) downloads.DatasetClient {
//	c := gomock.NewController()
//	return &mocks.cMockDatasetClient{}
//}

//var _ service.Dependencies = &External{}

type MyHandler func(w http.ResponseWriter, r http.Request)

func (m MyHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	panic("implement me")
}

var myHandler MyHandler

func NewDownloadServiceComponent(handler http.Handler, mongoUrl string) *DownloadServiceComponent {

	os.Setenv("MONGO_URL", mongoUrl)
	os.Setenv("DATABASE_NAME", "testing")

	log.Namespace = "dp-download-service"

	ctx := context.Background()

	cfg, err := config.Get()
	assert.NoError(&componenttest.ErrorFeature{}, err, "error getting config")

	log.Info(ctx, "config on startup", log.Data{"config": cfg})

	svc, err := service.New(ctx, "1", "1", "1", cfg, &external.External{})

	//fakeService := httpfake.New()

	//svc.Server = dphttp.NewServer("", fakeService.Server.Config.Handler)

	if err != nil {
		log.Fatal(ctx, "could not set up Download service", err)
		os.Exit(1)
	}
	fmt.Println("here1")


	svc.Run(ctx)

	fmt.Println("here2")

	return &DownloadServiceComponent{
		Handler: handler,
	}
}

func (m *DownloadServiceComponent) Initialiser(h http.Handler) componenttest.ServiceInitialiser {
	return func() (http.Handler, error) {
		m.Handler = h
		return h, nil
	}
}
