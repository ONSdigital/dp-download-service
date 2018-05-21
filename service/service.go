package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/justinas/alice"

	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/handlers"
	"github.com/ONSdigital/go-ns/clients/clientlog"
	"github.com/ONSdigital/go-ns/clients/dataset"
	"github.com/ONSdigital/go-ns/clients/filter"
	"github.com/ONSdigital/go-ns/healthcheck"
	"github.com/ONSdigital/go-ns/identity"
	"github.com/ONSdigital/go-ns/log"
	"github.com/ONSdigital/go-ns/rchttp"
	"github.com/ONSdigital/go-ns/server"
	"github.com/ONSdigital/s3crypto"
	"github.com/gorilla/mux"
	"github.com/ONSdigital/go-ns/common"
)

// Download represents the configuration to run the download service
type Download struct {
	datasetClient       DatasetClient
	filterClient        FilterClient
	vaultClient         VaultClient
	router              *mux.Router
	server              *server.Server
	errChan             chan error
	shutdown            time.Duration
	healthcheckInterval time.Duration
}

// DatasetClient is an interface to represent methods called to action on the dataset api
type DatasetClient interface {
	GetVersion(ctx context.Context, id, edition, version string) (m dataset.Version, err error)
	healthcheck.Client
}

// FilterClient is an interface to represent methods called to action on the filter api
type FilterClient interface {
	GetOutput(filterOutputID string, cfg ...filter.Config) (filter.Model, error)
	healthcheck.Client
}

// VaultClient is an interface to represent methods called to action upon vault
type VaultClient interface {
	ReadKey(path, key string) (string, error)
	healthcheck.Client
}

// Create should be called to create a new instance of the download service, with routes correctly initialised
func Create(bindAddr, vaultPath, bucketName, serviceToken, downloadServiceToken, zebedeeURL string,
	dc DatasetClient,
	fc FilterClient,
	s3sess *session.Session,
	vc VaultClient,
	shutdown, healthcheckInterval time.Duration,
	isPublishing bool) Download {

	router := mux.NewRouter()

	rchttpClient := rchttp.ClientWithServiceToken(rchttp.DefaultClient, serviceToken)
	rchttpClient.SetDownloadServiceToken(downloadServiceToken)
	filterClient := FilterClientImpl{rchttpClient}

	d := handlers.Download{
		DatasetClient: dc,
		VaultClient:   vc,
		FilterClient:  filterClient,
		S3Client:      s3crypto.New(s3sess, &s3crypto.Config{HasUserDefinedPSK: true}),
		BucketName:    bucketName,
		VaultPath:     vaultPath,
		IsPublishing:  isPublishing,
	}

	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv").HandlerFunc(d.Do("csv"))
	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.xlsx").HandlerFunc(d.Do("xls"))
	router.Path("/downloads/filter-outputs/{filterOutputID}.csv").HandlerFunc(d.Do("csv"))
	router.Path("/downloads/filter-outputs/{filterOutputID}.xlsx").HandlerFunc(d.Do("xls"))

	healthcheckHandler := healthcheck.NewMiddleware(healthcheck.Do)
	middlewareChain := alice.New(healthcheckHandler)

	if isPublishing {

		log.Debug("private endpoints are enabled. using identity middleware", nil)
		identityHandler := identity.Handler(zebedeeURL)
		middlewareChain = middlewareChain.Append(identityHandler)
	}

	alice := middlewareChain.Then(router)
	httpServer := server.New(bindAddr, alice)

	return Download{
		filterClient:        fc,
		datasetClient:       dc,
		router:              router,
		server:              httpServer,
		shutdown:            shutdown,
		healthcheckInterval: healthcheckInterval,
		errChan:             make(chan error, 1),
	}
}

// Start should be called to manage the running of the download service
func (d Download) Start() {
	healthTicker := healthcheck.NewTicker(d.healthcheckInterval, d.datasetClient, d.filterClient)
	d.server.HandleOSSignals = false

	d.run()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-d.errChan:
		log.ErrorC("download service error received", err, nil)
	case <-signals:
		log.Info("os signal received", nil)
	}

	// Gracefully shutdown the application closing any open resources.
	log.Info("shutdown with timeout", log.Data{"timeout": d.shutdown})
	ctx, cancel := context.WithTimeout(context.Background(), d.shutdown)

	start := time.Now()
	d.close(ctx)
	healthTicker.Close()

	log.Info("shutdown complete", log.Data{"duration": time.Since(start)})
	cancel()
	os.Exit(1)
}

func (d Download) run() {
	go func() {
		log.Debug("starting download service...", nil)
		if err := d.server.ListenAndServe(); err != nil {
			log.ErrorC("download service http service returned an error", err, nil)
			d.errChan <- err
		}
	}()
}

func (d Download) close(ctx context.Context) error {
	if err := d.server.Shutdown(ctx); err != nil {
		return err
	}
	log.Info("graceful shutdown of http server complete", nil)
	return nil
}

// FilterClientImpl implements FilterClient
// TODO: Switch this out for the updated go-ns client when it is available
type FilterClientImpl struct {
	client common.RCHTTPClienter
}

// GetOutput retrieves a filter output from the filter api
// TODO: Switch this out for the updated go-ns client when it is available
func (c FilterClientImpl) GetOutput(ctx context.Context, filterOutputID string) (m filter.Model, err error) {
	cfg, err := config.Get()
	if err != nil {
		return
	}

	uri := fmt.Sprintf("%s/filter-outputs/%s", cfg.FilterAPIURL, filterOutputID)

	clientlog.Do("retrieving filter output", "filter-api", uri)

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return
	}

	resp, err := c.client.Do(ctx, req)
	if err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		err = errors.New("unexpected filter api request")
		return
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	err = json.Unmarshal(b, &m)
	return
}
