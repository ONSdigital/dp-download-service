package service

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ONSdigital/dp-api-clients-go/health"
	clientsidentity "github.com/ONSdigital/dp-api-clients-go/identity"
	"github.com/ONSdigital/dp-api-clients-go/middleware"
	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/content"
	"github.com/ONSdigital/dp-download-service/downloads"
	"github.com/ONSdigital/dp-download-service/handlers"
	"github.com/justinas/alice"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	dphandlers "github.com/ONSdigital/dp-net/handlers"
	dphttp "github.com/ONSdigital/dp-net/http"
	"github.com/ONSdigital/log.go/log"
	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// Download represents the configuration to run the download service
type Download struct {
	datasetClient       downloads.DatasetClient
	filterClient        downloads.FilterClient
	imageClient         downloads.ImageClient
	vaultClient         content.VaultClient
	s3Client            content.S3Client
	zebedeeHealthClient *health.Client
	mongoClient         MongoClient
	router              *mux.Router
	server              *dphttp.Server
	shutdown            time.Duration
	healthCheck         HealthChecker
}

// Generate mocks of dependencies
//
//go:generate moq -rm -pkg service_test -out moq_service_test.go . Dependencies HealthChecker MongoClient
//go:generate moq -rm -pkg service_test -out moq_downloads_test.go ../downloads DatasetClient FilterClient ImageClient
//go:generate moq -rm -pkg service_test -out moq_content_test.go ../content S3Client VaultClient

// Dependencies holds constructors/factories for all external dependencies
//
type Dependencies interface {
	DatasetClient(string) downloads.DatasetClient
	FilterClient(string) downloads.FilterClient
	ImageClient(string) downloads.ImageClient
	VaultClient(*config.Config) (content.VaultClient, error)
	S3Client(*config.Config) (content.S3Client, error)
	MongoClient(context.Context, *config.Config) (MongoClient, error)
	HealthCheck(*config.Config, string, string, string) (HealthChecker, error)
}

// HealthChecker abstracts healthcheck.HealthCheck so we can create a mock.
// (interfaces for other dependencies are in ../downloads and ../content)
//
type HealthChecker interface {
	AddCheck(string, healthcheck.Checker) error
	Start(context.Context)
	Stop()
	Handler(http.ResponseWriter, *http.Request)
}

// Mongo abstracts mongo.Mongo so we can create a mock.
//
type MongoClient interface {
	URI() string
	Close(context.Context) error
	Checker(context.Context, *healthcheck.CheckState) error
}

// New returns a new Download service with dependencies initialised based on cfg and deps.
//
func New(ctx context.Context, buildTime, gitCommit, version string, cfg *config.Config, deps Dependencies) (*Download, error) {
	svc := &Download{
		datasetClient: deps.DatasetClient(cfg.DatasetAPIURL),
		filterClient:  deps.FilterClient(cfg.FilterAPIURL),
		imageClient:   deps.ImageClient(cfg.ImageAPIURL),
		shutdown:      cfg.GracefulShutdownTimeout,
	}

	// Vault client is set up only when encryption is disabled.
	//
	var err error
	var vc content.VaultClient
	if !cfg.EncryptionDisabled {
		vc, err = deps.VaultClient(cfg)
		if err != nil {
			log.Event(ctx, "could not create a vault client", log.FATAL, log.Error(err))
			return nil, err
		}
	}
	svc.vaultClient = vc

	// Set up S3 client.
	//
	s3, err := deps.S3Client(cfg)
	if err != nil {
		log.Event(ctx, "could not create the s3 client", log.ERROR, log.Error(err))
		return nil, err
	}
	svc.s3Client = s3

	// Only set up mongo when enabled with feature flag.
	//
	var mongoClient MongoClient
	if cfg.EnableMongo {
		mongoClient, err = deps.MongoClient(ctx, cfg)
		if err != nil {
			log.Event(ctx, "could not create mongo client", log.FATAL, log.Error(err))
			return nil, err
		}
		log.Event(ctx, "listening to mongo db session", log.INFO, log.Data{"URI": mongoClient.URI()})
	}
	svc.mongoClient = mongoClient

	// Create Health client for Zebedee only if we are in publishing mode.
	//
	var zc *health.Client
	if cfg.IsPublishing {
		zc = health.NewClient("Zebedee", cfg.ZebedeeURL)
	}
	svc.zebedeeHealthClient = zc

	// Set up health checkers for enabled dependencies.
	//
	hc, err := deps.HealthCheck(cfg, buildTime, gitCommit, version)
	if err != nil {
		log.Event(ctx, "could not create health checker", log.FATAL, log.Error(err))
		return nil, err
	}
	svc.healthCheck = hc
	if err = svc.registerCheckers(ctx); err != nil {
		return nil, err
	}

	// Set up download handler.
	//
	downloader := downloads.Downloader{
		FilterCli:  svc.filterClient,
		DatasetCli: svc.datasetClient,
		ImageCli:   svc.imageClient,
	}
	s3c := content.NewStreamWriter(s3, vc, cfg.VaultPath, cfg.EncryptionDisabled)

	d := handlers.Download{
		Downloader:   downloader,
		S3Content:    s3c,
		IsPublishing: cfg.IsPublishing,
	}

	// And tie routes to download hander methods.
	//
	router := mux.NewRouter()
	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv").HandlerFunc(d.DoDatasetVersion("csv", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv-metadata.json").HandlerFunc(d.DoDatasetVersion("csvw", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.xlsx").HandlerFunc(d.DoDatasetVersion("xls", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads/filter-outputs/{filterOutputID}.csv").HandlerFunc(d.DoFilterOutput("csv", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads/filter-outputs/{filterOutputID}.xlsx").HandlerFunc(d.DoFilterOutput("xls", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/images/{imageID}/{variant}/{filename}").HandlerFunc(d.DoImage(cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.HandleFunc("/health", hc.Handler)
	svc.router = router

	// Create new middleware chain with whitelisted handler for /health endpoint
	//
	middlewareChain := alice.New(middleware.Whitelist(middleware.HealthcheckFilter(hc.Handler)))

	// For non-whitelisted endpoints, do identityHandler or corsHandler
	//
	if cfg.IsPublishing {
		log.Event(ctx, "private endpoints are enabled. using identity middleware", log.INFO)
		identityHandler := dphandlers.IdentityWithHTTPClient(clientsidentity.NewWithHealthClient(zc))
		middlewareChain = middlewareChain.Append(identityHandler)
	} else {
		corsHandler := gorillahandlers.CORS(gorillahandlers.AllowedMethods([]string{"GET"}))
		middlewareChain = middlewareChain.Append(corsHandler)
	}

	r := middlewareChain.
		Append(dphandlers.CheckHeader(dphandlers.UserAccess)).
		Append(dphandlers.CheckHeader(dphandlers.CollectionID)).
		Then(router)
	svc.server = dphttp.NewServer(cfg.BindAddr, r)

	return svc, nil
}

func (svc *Download) registerCheckers(ctx context.Context) error {
	var hasErrors bool
	hc := svc.healthCheck

	if err := hc.AddCheck("Dataset API", svc.datasetClient.Checker); err != nil {
		hasErrors = true
		log.Event(ctx, "error adding check for dataset api", log.ERROR, log.Error(err))
	}

	if svc.vaultClient != nil {
		if err := hc.AddCheck("Vault", svc.vaultClient.Checker); err != nil {
			hasErrors = true
			log.Event(ctx, "error adding check for vault", log.ERROR, log.Error(err))
		}
	}

	if err := hc.AddCheck("Filter API", svc.filterClient.Checker); err != nil {
		hasErrors = true
		log.Event(ctx, "error adding check for filter api", log.ERROR, log.Error(err))
	}

	if err := hc.AddCheck("Image API", svc.imageClient.Checker); err != nil {
		hasErrors = true
		log.Event(ctx, "error adding check for image api", log.ERROR, log.Error(err))
	}

	if svc.zebedeeHealthClient != nil {
		if err := hc.AddCheck("Zebedee", svc.zebedeeHealthClient.Checker); err != nil {
			hasErrors = true
			log.Event(ctx, "error adding check for zebedee", log.ERROR, log.Error(err))
		}
	}

	if err := hc.AddCheck("S3", svc.s3Client.Checker); err != nil {
		hasErrors = true
		log.Event(ctx, "error adding check for s3", log.ERROR, log.Error(err))
	}

	if svc.mongoClient != nil {
		if err := hc.AddCheck("Mongo", svc.mongoClient.Checker); err != nil {
			hasErrors = true
			log.Event(ctx, "error adding check for mongo", log.ERROR, log.Error(err))
		}
	}

	if hasErrors {
		return errors.New("Error(s) registering checkers for healthcheck")
	}
	return nil
}

// Start should be called to manage the running of the download service
func (d Download) Start(ctx context.Context) {

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	d.server.HandleOSSignals = false

	d.healthCheck.Start(ctx)
	d.run(ctx)

	<-signals
	log.Event(ctx, "os signal received", log.INFO)

	shutdownCtx, cancel := context.WithTimeout(ctx, d.shutdown)

	// Gracefully shutdown the application closing any open resources.
	log.Event(shutdownCtx, "shutdown with timeout", log.INFO, log.Data{"timeout": d.shutdown})

	shutdownStart := time.Now()
	d.close(shutdownCtx)
	d.healthCheck.Stop()

	log.Event(shutdownCtx, "shutdown complete", log.INFO, log.Data{"duration": time.Since(shutdownStart)})
	cancel()
	os.Exit(1)
}

func (d Download) run(ctx context.Context) {
	go func() {
		log.Event(ctx, "starting download service...", log.INFO)
		if err := d.server.ListenAndServe(); err != nil {
			log.Event(ctx, "download service http service returned an error", log.ERROR, log.Error(err))
		}
	}()
}

func (d Download) close(ctx context.Context) error {
	if err := d.server.Shutdown(ctx); err != nil {
		return err
	}
	log.Event(ctx, "graceful shutdown of http server complete", log.INFO)
	return nil
}
