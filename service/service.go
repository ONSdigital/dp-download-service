package service

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/justinas/alice"

	"github.com/ONSdigital/dp-api-clients-go/dataset"
	"github.com/ONSdigital/dp-api-clients-go/filter"
	"github.com/ONSdigital/dp-download-service/handlers"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/go-ns/identity"
	"github.com/ONSdigital/go-ns/server"
	"github.com/ONSdigital/log.go/log"
	"github.com/ONSdigital/s3crypto"
	"github.com/gorilla/mux"

	gorillahandlers "github.com/gorilla/handlers"
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
	healthCheckInterval time.Duration
	healthCheckRecovery time.Duration
	healthCheck         *healthcheck.HealthCheck
}

// DatasetClient is an interface to represent methods called to action on the dataset api
type DatasetClient interface {
	GetVersion(ctx context.Context, userAuthToken, serviceAuthToken, downloadServiceAuthToken, collectionID, datasetID, edition, version string) (m dataset.Version, err error)
	// healthcheck.Client
}

// FilterClient is an interface to represent methods called to action on the filter api
type FilterClient interface {
	GetOutput(ctx context.Context, userAuthToken, serviceAuthToken, downloadServiceAuthToken, collectionID, filterOutputID string) (filter.Model, error)
	// healthcheck.Client
}

// VaultClient is an interface to represent methods called to action upon vault
type VaultClient interface {
	ReadKey(path, key string) (string, error)
	// healthcheck.Client
}

// Create should be called to create a new instance of the download service, with routes correctly initialised
func Create(bindAddr, vaultPath, bucketName, serviceAuthToken, downloadServiceToken, zebedeeURL string,
	dc DatasetClient,
	fc FilterClient,
	s3sess *session.Session,
	vc VaultClient,
	shutdown, healthCheckInterval, healthCheckRecovery time.Duration,
	isPublishing bool,
	buildTime, gitCommit, version string) Download {

	ctx := context.Background()
	router := mux.NewRouter()

	d := handlers.Download{
		DatasetClient: dc,
		VaultClient:   vc,
		FilterClient:  fc,
		S3Client:      s3crypto.New(s3sess, &s3crypto.Config{HasUserDefinedPSK: true}),
		BucketName:    bucketName,
		VaultPath:     vaultPath,
		IsPublishing:  isPublishing,
	}

	// Create healthcheck object with versionInfo
	versionInfo, err := healthcheck.NewVersionInfo(buildTime, gitCommit, version)
	if err != nil {
		log.Event(ctx, "initialising healthcheck without versionInfo", log.Error(err))
	}
	hc := healthcheck.New(versionInfo, healthCheckRecovery, healthCheckInterval)

	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv").HandlerFunc(d.Do("csv", serviceAuthToken, downloadServiceToken))
	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv-metadata.json").HandlerFunc(d.Do("csvw", serviceAuthToken, downloadServiceToken))
	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.xlsx").HandlerFunc(d.Do("xls", serviceAuthToken, downloadServiceToken))
	router.Path("/downloads/filter-outputs/{filterOutputID}.csv").HandlerFunc(d.Do("csv", serviceAuthToken, downloadServiceToken))
	router.Path("/downloads/filter-outputs/{filterOutputID}.xlsx").HandlerFunc(d.Do("xls", serviceAuthToken, downloadServiceToken))
	router.HandleFunc("/health", hc.Handler)

	middlewareChain := alice.New()

	if isPublishing {
		log.Event(ctx, "private endpoints are enabled. using identity middleware")
		identityHandler := identity.Handler(zebedeeURL)
		middlewareChain = middlewareChain.Append(identityHandler)
	} else {
		corsHandler := gorillahandlers.CORS(gorillahandlers.AllowedMethods([]string{"GET"}))
		middlewareChain = middlewareChain.Append(corsHandler)
	}

	alice := middlewareChain.Then(router)
	httpServer := server.New(bindAddr, alice)

	return Download{
		filterClient:        fc,
		datasetClient:       dc,
		router:              router,
		server:              httpServer,
		shutdown:            shutdown,
		healthCheckInterval: healthCheckInterval,
		healthCheckRecovery: healthCheckRecovery,
		errChan:             make(chan error, 1),
		healthCheck:         &hc,
	}
}

// Start should be called to manage the running of the download service
func (d Download) Start() {

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	d.server.HandleOSSignals = false
	ctx, cancel := context.WithTimeout(context.Background(), d.shutdown)

	// Start Healthcheck and server
	d.healthCheck.Start(ctx)
	d.run()

	select {
	case err := <-d.errChan:
		log.Event(ctx, "download service error received", log.Error(err))
	case <-signals:
		log.Event(ctx, "os signal received")
	}

	// Gracefully shutdown the application closing any open resources.
	log.Event(ctx, "shutdown with timeout", log.Data{"timeout": d.shutdown})

	shutdownStart := time.Now()
	d.close(ctx)
	d.healthCheck.Stop()

	log.Event(ctx, "shutdown complete", log.Data{"duration": time.Since(shutdownStart)})
	cancel()
	os.Exit(1)
}

func (d Download) run() {
	ctx := context.Background()
	go func() {
		log.Event(ctx, "starting download service...")
		if err := d.server.ListenAndServe(); err != nil {
			log.Event(ctx, "download service http service returned an error", log.Error(err))
			d.errChan <- err
		}
	}()
}

func (d Download) close(ctx context.Context) error {
	if err := d.server.Shutdown(ctx); err != nil {
		return err
	}
	log.Event(ctx, "graceful shutdown of http server complete")
	return nil
}
