package steps

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/maxcnunes/httpfake"

	componenttest "github.com/ONSdigital/dp-component-test"
	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/service"
	dphttp "github.com/ONSdigital/dp-net/v2/http"
	"github.com/ONSdigital/log.go/v2/log"
)

const (
	localStackHost = "http://localstack:4566"
)

type DownloadServiceComponent struct {
	DpHttpServer *dphttp.Server
	svc          *service.Download
	ApiFeature   *componenttest.APIFeature
	errChan      chan error
}

func NewDownloadServiceComponent(fake_auth_url string) *DownloadServiceComponent {
	//os.Setenv("ZEBEDEE_URL", fake_auth_url)

	d := &DownloadServiceComponent{
		DpHttpServer: dphttp.NewServer("", http.NewServeMux()),
		errChan:      make(chan error),
	}

	os.Setenv("DATABASE_NAME", "testing")

	log.Namespace = "dp-download-service"

	//assert.NoError(&componenttest.ErrorFeature{}, err, "error getting config")

	return d
}

func (d *DownloadServiceComponent) Initialiser() (http.Handler, error) {
	fakeService := httpfake.New()
	fakeService.NewHandler().Get("/health").Reply(http.StatusOK)
	os.Setenv("ZEBEDEE_URL", fakeService.ResolveURL(""))
	cfg, _ := config.Get()
	d.svc, _ = service.New(context.Background(), "1", "1", "1", cfg, &External{Server: d.DpHttpServer})
	d.svc.Run(context.Background())
	time.Sleep(15 * time.Second)

	return d.DpHttpServer.Handler, nil
}
