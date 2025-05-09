package service

import (
	"context"
	"errors"
	"net/http"
	"time"

	clientsidentity "github.com/ONSdigital/dp-api-clients-go/v2/identity"
	"github.com/ONSdigital/dp-download-service/api"
	"github.com/ONSdigital/dp-download-service/files"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/ONSdigital/dp-api-clients-go/v2/health"
	"github.com/ONSdigital/dp-api-clients-go/v2/middleware"
	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/content"
	"github.com/ONSdigital/dp-download-service/downloads"
	"github.com/ONSdigital/dp-download-service/handlers"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	dphandlers "github.com/ONSdigital/dp-net/v2/handlers"
	"github.com/ONSdigital/log.go/v2/log"
	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
)

// Download represents the configuration to run the download service
type Download struct {
	datasetClient       downloads.DatasetClient
	filterClient        downloads.FilterClient
	imageClient         downloads.ImageClient
	filesClient         downloads.FilesClient
	s3Client            content.S3Client
	zebedeeHealthClient *health.Client
	router              *mux.Router
	server              HTTPServer
	shutdown            time.Duration
	healthCheck         HealthChecker
}

// Generate mocks of dependencies
//
//go:generate moq -pkg service_test -out moq_service_test.go . Dependencies HealthChecker HTTPServer
//go:generate moq -pkg service_test -out moq_downloads_test.go ../downloads DatasetClient FilterClient ImageClient FilesClient
//go:generate moq -pkg service_test -out moq_content_test.go ../content S3Client

// Dependencies holds constructors/factories for all external dependencies
//

type Dependencies interface {
	DatasetClient(string) downloads.DatasetClient
	FilterClient(string) downloads.FilterClient
	ImageClient(string) downloads.ImageClient
	S3Client(context.Context, *config.Config) (content.S3Client, error)
	FilesClient(*config.Config) downloads.FilesClient
	HealthCheck(*config.Config, string, string, string) (HealthChecker, error)
	HttpServer(*config.Config, http.Handler) HTTPServer
}

// HealthChecker abstracts healthcheck.HealthCheck so we can create a mock.
// (interfaces for other dependencies are in ../downloads and ../content)
type HealthChecker interface {
	AddCheck(string, healthcheck.Checker) error
	Start(context.Context)
	Stop()
	Handler(http.ResponseWriter, *http.Request)
}

// HTTPServer defines the required methods from the HTTP server
type HTTPServer interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

// New returns a new Download service with dependencies initialised based on cfg and deps.
func New(ctx context.Context, buildTime, gitCommit, version string, cfg *config.Config, deps Dependencies) (*Download, error) {
	svc := &Download{
		datasetClient: deps.DatasetClient(cfg.DatasetAPIURL),
		filterClient:  deps.FilterClient(cfg.FilterAPIURL),
		imageClient:   deps.ImageClient(cfg.ImageAPIURL),
		filesClient:   deps.FilesClient(cfg),
		shutdown:      cfg.GracefulShutdownTimeout,
	}

	var err error

	// Set up S3 client.
	//
	s3, err := deps.S3Client(ctx, cfg)
	if err != nil {
		log.Error(ctx, "could not create the s3 client", err)
		return nil, err
	}
	svc.s3Client = s3

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
		log.Fatal(ctx, "could not create health checker", err)
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
	s3c := content.NewStreamWriter(s3)

	d := handlers.Download{
		Downloader:   downloader,
		S3Content:    s3c,
		IsPublishing: cfg.IsPublishing,
	}

	// Flagged off? Assumption that downloads-new is related to uploads-new
	downloadHandler := api.CreateV1DownloadHandler(
		files.FetchMetadata(svc.filesClient, cfg.ServiceAuthToken),
		files.DownloadFile(ctx, svc.s3Client),
		cfg,
	)

	// The 'Do' functions eventually get to the S3 bucket, which is all of them except the V1 downloader
	// And tie routes to download handler methods.
	//
	router := mux.NewRouter()
	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv").HandlerFunc(d.DoDatasetVersion("csv", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv-metadata.json").HandlerFunc(d.DoDatasetVersion("csvw", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.txt").HandlerFunc(d.DoDatasetVersion("txt", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.xls").HandlerFunc(d.DoDatasetVersion("xls", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.xlsx").HandlerFunc(d.DoDatasetVersion("xls", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads/filter-outputs/{filterOutputID}.csv").HandlerFunc(d.DoFilterOutput("csv", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads/filter-outputs/{filterOutputID}.xls").HandlerFunc(d.DoFilterOutput("xls", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads/filter-outputs/{filterOutputID}.xlsx").HandlerFunc(d.DoFilterOutput("xls", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads/filter-outputs/{filterOutputID}.txt").HandlerFunc(d.DoFilterOutput("txt", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads/filter-outputs/{filterOutputID}.csv-metadata.json").HandlerFunc(d.DoFilterOutput("csvw", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/images/{imageID}/{variant}/{filename}").HandlerFunc(d.DoImage(cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads-new/{path:.*}").HandlerFunc(downloadHandler)
	router.HandleFunc("/health", hc.Handler)
	svc.router = router

	// Create new middleware chain with whitelisted handler for /health endpoint
	//
	middlewareChain := alice.New(middleware.Whitelist(middleware.HealthcheckFilter(hc.Handler)))
	middlewareChain = middlewareChain.Append(api.Limiter(cfg.MaxConcurrentHandlers))

	if cfg.OtelEnabled {
		// Add middleware for open telemetry
		router.Use(otelmux.Middleware(cfg.OTServiceName))
		middlewareChain = middlewareChain.Append(otelhttp.NewMiddleware(cfg.OTServiceName))
	}
	// For non-whitelisted endpoints, do identityHandler or corsHandler
	//
	if cfg.IsPublishing {
		log.Info(ctx, "private endpoints are enabled. using identity middleware")
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

	svc.server = deps.HttpServer(cfg, r)

	return svc, nil
}

func (svc *Download) registerCheckers(ctx context.Context) error {
	var hasErrors bool
	hc := svc.healthCheck

	if err := hc.AddCheck("Dataset API", svc.datasetClient.Checker); err != nil {
		hasErrors = true
		log.Error(ctx, "error adding check for dataset api", err)
	}

	if err := hc.AddCheck("Filter API", svc.filterClient.Checker); err != nil {
		hasErrors = true
		log.Error(ctx, "error adding check for filter api", err)
	}

	if err := hc.AddCheck("Image API", svc.imageClient.Checker); err != nil {
		hasErrors = true
		log.Error(ctx, "error adding check for image api", err)
	}

	if svc.zebedeeHealthClient != nil {
		if err := hc.AddCheck("Zebedee", svc.zebedeeHealthClient.Checker); err != nil {
			hasErrors = true
			log.Error(ctx, "error adding check for zebedee", err)
		}
	}

	if err := hc.AddCheck("S3", svc.s3Client.Checker); err != nil {
		hasErrors = true
		log.Error(ctx, "error adding check for s3", err)
	}

	if hasErrors {
		return errors.New("Error(s) registering checkers for healthcheck")
	}
	return nil
}

func (d Download) Run(ctx context.Context) {
	//d.server.HandleOSSignals = false

	d.healthCheck.Start(ctx)
	go func() {
		log.Info(ctx, "starting download service...")
		if err := d.server.ListenAndServe(); err != nil {
			log.Error(ctx, "download service http service returned an error", err)
		}
	}()
}

func (d Download) Close(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, d.shutdown)
	defer cancel()

	// Gracefully shutdown the application closing any open resources.
	log.Info(shutdownCtx, "shutdown with timeout", log.Data{"timeout": d.shutdown})

	shutdownStart := time.Now()
	d.healthCheck.Stop()

	if err := d.server.Shutdown(ctx); err != nil {
		return err
	}

	log.Info(shutdownCtx, "shutdown complete", log.Data{"duration": time.Since(shutdownStart)})

	return nil
}
