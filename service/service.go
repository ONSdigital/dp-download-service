package service

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/justinas/alice"

	"github.com/ONSdigital/dp-api-clients-go/health"
	clientsidentity "github.com/ONSdigital/dp-api-clients-go/identity"
	"github.com/ONSdigital/dp-api-clients-go/middleware"
	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/handlers"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/go-ns/identity"
	"github.com/ONSdigital/go-ns/server"
	"github.com/ONSdigital/log.go/log"
	"github.com/gorilla/mux"

	gorillahandlers "github.com/gorilla/handlers"
)

// Download represents the configuration to run the download service
type Download struct {
	datasetClient handlers.DatasetClient
	filterClient  handlers.FilterClient
	vaultClient   handlers.VaultClient
	router        *mux.Router
	server        *server.Server
	errChan       chan error
	shutdown      time.Duration
	healthCheck   *healthcheck.HealthCheck
}

// Create should be called to create a new instance of the download service, with routes correctly initialised.
// Note: zc is allowed to be nil if we are not in publishing mode
func Create(
	cfg config.Config,
	dc handlers.DatasetClient,
	fc handlers.FilterClient,
	s3 handlers.S3Client,
	vc handlers.VaultClient,
	zc *health.Client,
	hc *healthcheck.HealthCheck) Download {

	ctx := context.Background()
	router := mux.NewRouter()

	d := handlers.Download{
		DatasetClient: dc,
		VaultClient:   vc,
		FilterClient:  fc,
		S3Client:      s3,
		VaultPath:     cfg.VaultPath,
		IsPublishing:  cfg.IsPublishing,
	}

	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv").HandlerFunc(d.Do("csv", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv-metadata.json").HandlerFunc(d.Do("csvw", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.xlsx").HandlerFunc(d.Do("xls", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads/filter-outputs/{filterOutputID}.csv").HandlerFunc(d.Do("csv", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.Path("/downloads/filter-outputs/{filterOutputID}.xlsx").HandlerFunc(d.Do("xls", cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	router.HandleFunc("/health", hc.Handler)

	// Create new middleware chain with whitelisted handler for /health endpoint
	middlewareChain := alice.New(middleware.Whitelist(middleware.HealthcheckFilter(hc.Handler)))

	// For non-whitelisted endpoints, do identityHandler or corsHandler
	if cfg.IsPublishing {
		log.Event(ctx, "private endpoints are enabled. using identity middleware")
		identityHandler := identity.HandlerForHTTPClient(clientsidentity.NewAPIClient(zc.Client, cfg.ZebedeeURL))
		middlewareChain = middlewareChain.Append(identityHandler)
	} else {
		corsHandler := gorillahandlers.CORS(gorillahandlers.AllowedMethods([]string{"GET"}))
		middlewareChain = middlewareChain.Append(corsHandler)
	}

	alice := middlewareChain.Then(router)
	httpServer := server.New(cfg.BindAddr, alice)

	return Download{
		filterClient:  fc,
		datasetClient: dc,
		vaultClient:   vc,
		router:        router,
		server:        httpServer,
		shutdown:      cfg.GracefulShutdownTimeout,
		errChan:       make(chan error, 1),
		healthCheck:   hc,
	}
}

// Start should be called to manage the running of the download service
func (d Download) Start() {

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	d.server.HandleOSSignals = false
	hcCtx := context.Background()
	ctx, cancel := context.WithTimeout(hcCtx, d.shutdown)

	d.healthCheck.Start(hcCtx)
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
