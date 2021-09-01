package service

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ONSdigital/dp-download-service/content"
	"github.com/ONSdigital/dp-download-service/downloads"
	"github.com/justinas/alice"

	"github.com/ONSdigital/dp-api-clients-go/health"
	clientsidentity "github.com/ONSdigital/dp-api-clients-go/identity"
	"github.com/ONSdigital/dp-api-clients-go/middleware"
	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/handlers"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	dphandlers "github.com/ONSdigital/dp-net/handlers"
	dphttp "github.com/ONSdigital/dp-net/http"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/gorilla/mux"

	gorillahandlers "github.com/gorilla/handlers"
)

// Download represents the configuration to run the download service
type Download struct {
	datasetClient downloads.DatasetClient
	filterClient  downloads.FilterClient
	imageClient   downloads.ImageClient
	vaultClient   content.VaultClient
	router        *mux.Router
	server        *dphttp.Server
	shutdown      time.Duration
	healthCheck   *healthcheck.HealthCheck
}

// Create should be called to create a new instance of the download service, with routes correctly initialised.
// Note: zc is allowed to be nil if we are not in publishing mode
func Create(
	ctx context.Context,
	cfg config.Config,
	dc downloads.DatasetClient,
	fc downloads.FilterClient,
	ic downloads.ImageClient,
	s3 content.S3Client,
	vc content.VaultClient,
	zc *health.Client,
	hc *healthcheck.HealthCheck) Download {

	router := mux.NewRouter()

	downloader := downloads.Downloader{
		FilterCli:  fc,
		DatasetCli: dc,
		ImageCli:   ic,
	}

	s3c := content.NewStreamWriter(s3, vc, cfg.VaultPath, cfg.EncryptionDisabled)

	d := handlers.Download{
		Downloader:   downloader,
		S3Content:    s3c,
		IsPublishing: cfg.IsPublishing,
	}

	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv").HandlerFunc(d.DoDatasetVersion("csv", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv-metadata.json").HandlerFunc(d.DoDatasetVersion("csvw", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.xlsx").HandlerFunc(d.DoDatasetVersion("xls", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads/filter-outputs/{filterOutputID}.csv").HandlerFunc(d.DoFilterOutput("csv", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads/filter-outputs/{filterOutputID}.xlsx").HandlerFunc(d.DoFilterOutput("xls", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/images/{imageID}/{variant}/{filename}").HandlerFunc(d.DoImage(cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.HandleFunc("/health", hc.Handler)

	// Create new middleware chain with whitelisted handler for /health endpoint
	middlewareChain := alice.New(middleware.Whitelist(middleware.HealthcheckFilter(hc.Handler)))

	// For non-whitelisted endpoints, do identityHandler or corsHandler
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
	httpServer := dphttp.NewServer(cfg.BindAddr, r)

	return Download{
		filterClient:  fc,
		imageClient:   ic,
		datasetClient: dc,
		vaultClient:   vc,
		router:        router,
		server:        httpServer,
		shutdown:      cfg.GracefulShutdownTimeout,
		healthCheck:   hc,
	}
}

// Start should be called to manage the running of the download service
func (d Download) Start(ctx context.Context) {

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	d.server.HandleOSSignals = false

	d.healthCheck.Start(ctx)
	d.run(ctx)

	<-signals
	log.Info(ctx, "os signal received")

	shutdownCtx, cancel := context.WithTimeout(ctx, d.shutdown)

	// Gracefully shutdown the application closing any open resources.
	log.Info(shutdownCtx, "shutdown with timeout", log.Data{"timeout": d.shutdown})

	shutdownStart := time.Now()
	d.close(shutdownCtx)
	d.healthCheck.Stop()

	log.Info(shutdownCtx, "shutdown complete", log.Data{"duration": time.Since(shutdownStart)})
	cancel()
	os.Exit(1)
}

func (d Download) run(ctx context.Context) {
	go func() {
		log.Info(ctx, "starting download service...")
		if err := d.server.ListenAndServe(); err != nil {
			log.Error(ctx, "download service http service returned an error", err)
		}
	}()
}

func (d Download) close(ctx context.Context) error {
	if err := d.server.Shutdown(ctx); err != nil {
		return err
	}
	log.Info(ctx, "graceful shutdown of http server complete")
	return nil
}
