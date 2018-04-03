package service

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/ONSdigital/dp-download-service/handlers"
	"github.com/ONSdigital/go-ns/clients/dataset"
	"github.com/ONSdigital/go-ns/healthcheck"
	"github.com/ONSdigital/go-ns/log"
	"github.com/ONSdigital/go-ns/server"
	"github.com/ONSdigital/s3crypto"
	"github.com/gorilla/mux"
)

// Download represents the configuration to run the download service
type Download struct {
	datasetClient       DatasetClient
	vaultClient         VaultClient
	router              *mux.Router
	server              *server.Server
	errChan             chan error
	shutdown            time.Duration
	healthcheckInterval time.Duration
}

// DatasetClient is an interface to represent methods called to action on the dataset api
type DatasetClient interface {
	GetVersion(id, edition, version string, cfg ...dataset.Config) (m dataset.Version, err error)
	healthcheck.Client
}

// VaultClient is an interface to represent methods called to action upon vault
type VaultClient interface {
	ReadKey(path, key string) (string, error)
	healthcheck.Client
}

// Create should be called to create a new instance of the download service, with routes correctly initialised
func Create(bindAddr, secretKey, datasetAuthToken, xDownloadServiceAuthToken, vaultPath, bucketName, serviceToken string, dc DatasetClient, s3sess *session.Session, vc VaultClient, shutdown, healthcheckInterval time.Duration) Download {
	router := mux.NewRouter()

	d := handlers.Download{
		DatasetClient:             dc,
		VaultClient:               vc,
		S3Client:                  s3crypto.New(s3sess, &s3crypto.Config{HasUserDefinedPSK: true}),
		DatasetAuthToken:          datasetAuthToken,
		XDownloadServiceAuthToken: xDownloadServiceAuthToken,
		SecretKey:                 secretKey,
		BucketName:                bucketName,
		ServiceToken:              serviceToken,
		VaultPath:                 vaultPath,
	}

	router.Path("/healthcheck").Methods("GET").HandlerFunc(healthcheck.Do)
	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv").HandlerFunc(d.Do("csv"))
	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.xlsx").HandlerFunc(d.Do("xls"))

	return Download{
		datasetClient:       dc,
		router:              router,
		server:              server.New(bindAddr, router),
		shutdown:            shutdown,
		healthcheckInterval: healthcheckInterval,
		errChan:             make(chan error, 1),
	}
}

// Start should be called to manage the running of the download service
func (d Download) Start() {
	healthTicker := healthcheck.NewTicker(d.healthcheckInterval, d.datasetClient)
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
