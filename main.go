package main

import (
	"context"
	"errors"
	"os"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	s3client "github.com/ONSdigital/dp-s3"
	vault "github.com/ONSdigital/dp-vault"

	"github.com/ONSdigital/dp-api-clients-go/dataset"
	"github.com/ONSdigital/dp-api-clients-go/filter"
	"github.com/ONSdigital/dp-api-clients-go/health"
	"github.com/ONSdigital/dp-api-clients-go/image"
	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/mongo"
	"github.com/ONSdigital/dp-download-service/service"
	"github.com/ONSdigital/log.go/log"
)

var (
	// BuildTime represents the time in which the service was built
	BuildTime string
	// GitCommit represents the commit (SHA-1) hash of the service that is running
	GitCommit string
	// Version represents the version of the service that is running
	Version string
)

func main() {
	log.Namespace = "dp-download-service"

	ctx := context.Background()

	cfg, err := config.Get()
	if err != nil {
		log.Event(ctx, "error getting config", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	log.Event(ctx, "config on startup", log.INFO, log.Data{"config": cfg})

	// Create Dataset API client.
	dc := dataset.NewAPIClient(cfg.DatasetAPIURL)

	// Create Vault client.
	var vc *vault.Client = nil
	if !cfg.EncryptionDisabled {
		vc, err = vault.CreateClient(cfg.VaultToken, cfg.VaultAddress, 3)
		if err != nil {
			log.Event(ctx, "could not create a vault client", log.FATAL, log.Error(err))
			os.Exit(1)
		}
	}

	// Create Filter API client.
	fc := filter.New(cfg.FilterAPIURL)

	// Create Image API client.
	ic := image.NewAPIClient(cfg.ImageAPIURL)

	// Create Health client for Zebedee only if we are in publishing mode.
	var zc *health.Client
	if cfg.IsPublishing {
		zc = health.NewClient("Zebedee", cfg.ZebedeeURL)
	}

	// Create S3 client with region and bucket name.
	s3, err := s3client.NewClient(cfg.AwsRegion, cfg.BucketName, !cfg.EncryptionDisabled)
	if err != nil {
		log.Event(ctx, "could not create the s3 client", log.ERROR, log.Error(err))
	}

	// Create Mongo client.
	var mongodb *mongo.Mongo
	if cfg.EnableMongo {
		mongodb = &mongo.Mongo{
			Collection: cfg.MongoConfig.Collection,
			Database:   cfg.MongoConfig.Database,
			DatasetURL: cfg.DatasetAPIURL,
			URI:        cfg.MongoConfig.BindAddr,
			Username:   cfg.MongoConfig.Username,
			Password:   cfg.MongoConfig.Password,
			IsSSL:      cfg.MongoConfig.IsSSL,
		}
		if err = mongodb.Init(ctx); err != nil {
			log.Event(ctx, "could not create mongo client", log.FATAL, log.Error(err))
			os.Exit(1)
		}
		log.Event(ctx, "listening to mongo db session", log.INFO, log.Data{"URI": mongodb.URI})
	}

	// Create healthcheck object with versionInfo and register Checkers.
	versionInfo, err := healthcheck.NewVersionInfo(BuildTime, GitCommit, Version)
	if err != nil {
		log.Event(ctx, "failed to obtain version info for healthcheck", log.FATAL, log.Error(err))
		os.Exit(1)
	}
	hc := healthcheck.New(versionInfo, cfg.HealthCheckCriticalTimeout, cfg.HealthCheckInterval)
	if err = registerCheckers(ctx, &hc, cfg.IsPublishing, dc, vc, fc, ic, zc, s3, mongodb); err != nil {
		os.Exit(1)
	}

	// Create and start Service providing the required clients.
	svc := service.Create(
		ctx,
		*cfg,
		dc,
		fc,
		ic,
		s3,
		vc,
		zc,
		&hc,
	)
	svc.Start(ctx)
}

// registerCheckers adds the checkers for the provided clients to the healthcheck object.
// Zebedee health client will only be registered if we are in publishing mode.
func registerCheckers(ctx context.Context, hc *healthcheck.HealthCheck, isPublishing bool,
	dc *dataset.Client,
	vc *vault.Client,
	fc *filter.Client,
	ic *image.Client,
	zc *health.Client,
	s3 *s3client.S3,
	mongodb *mongo.Mongo) error {

	hasErrors := false

	if err := hc.AddCheck("Dataset API", dc.Checker); err != nil {
		hasErrors = true
		log.Event(ctx, "error adding check for dataset api", log.ERROR, log.Error(err))
	}

	if vc != nil {
		if err := hc.AddCheck("Vault", vc.Checker); err != nil {
			hasErrors = true
			log.Event(ctx, "error adding check for vault", log.ERROR, log.Error(err))
		}
	}

	if err := hc.AddCheck("Filter API", fc.Checker); err != nil {
		hasErrors = true
		log.Event(ctx, "error adding check for filter api", log.ERROR, log.Error(err))
	}

	if err := hc.AddCheck("Image API", ic.Checker); err != nil {
		hasErrors = true
		log.Event(ctx, "error adding check for image api", log.ERROR, log.Error(err))
	}

	if isPublishing {
		if err := hc.AddCheck("Zebedee", zc.Checker); err != nil {
			hasErrors = true
			log.Event(ctx, "error adding check for zebedee", log.ERROR, log.Error(err))
		}
	}

	if err := hc.AddCheck("S3", s3.Checker); err != nil {
		hasErrors = true
		log.Event(ctx, "error adding check for s3", log.ERROR, log.Error(err))
	}

	if mongodb != nil {
		if err := hc.AddCheck("Mongo", mongodb.Checker); err != nil {
			hasErrors = true
			log.Event(ctx, "error adding check for mongo", log.ERROR, log.Error(err))
		}
	}

	if hasErrors {
		return errors.New("Error(s) registering checkers for healthcheck")
	}
	return nil
}
