package main

import (
	"context"
	"os"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	vault "github.com/ONSdigital/dp-vault"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/ONSdigital/dp-api-clients-go/dataset"
	"github.com/ONSdigital/dp-api-clients-go/filter"
	"github.com/ONSdigital/dp-api-clients-go/health"
	"github.com/ONSdigital/dp-download-service/config"
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
		log.Event(ctx, "error getting config", log.Error(err))
		os.Exit(1)
	}

	log.Event(ctx, "config on startup", log.Data{"config": cfg})

	dc := dataset.NewAPIClient(cfg.DatasetAPIURL)
	vc, err := vault.CreateClient(cfg.VaultToken, cfg.VaultAddress, 3)
	if err != nil {
		log.Event(ctx, "could not create a vault client", log.Error(err))
		os.Exit(1)
	}

	fc := filter.New(cfg.FilterAPIURL)

	// Create Health client for Zebedee, if we are in publishing mode.
	var zc *health.Client
	if cfg.IsPublishing {
		zc = health.NewClient("Zebedee", cfg.ZebedeeURL)
	}

	// TODO migrate to dp-s3
	region := "eu-west-1"
	sess := session.New(&aws.Config{Region: &region})

	// Create healthcheck object with versionInfo
	versionInfo, err := healthcheck.NewVersionInfo(BuildTime, GitCommit, Version)
	if err != nil {
		log.Event(ctx, "Failed to obtain VersionInfo for healthcheck", log.Error(err))
		os.Exit(1)
	}
	hc := healthcheck.New(versionInfo, cfg.HealthCheckCriticalTimeout, cfg.HealthCheckInterval)
	registerCheckers(&hc, cfg.IsPublishing, dc, vc, fc, zc)

	svc := service.Create(
		*cfg,
		dc,
		fc,
		sess,
		vc,
		zc,
		&hc,
	)

	svc.Start()
}

// registerCheckers adds the checkers for the provided clients to the healthcheck object.
// Zebedee health client will only be registered if we are in publishing mode.
func registerCheckers(hc *healthcheck.HealthCheck, isPublishing bool,
	dc *dataset.Client,
	vc *vault.Client,
	fc *filter.Client,
	zc *health.Client) (err error) {

	if err = hc.AddCheck("Dataset API", dc.Checker); err != nil {
		log.Event(nil, "Error Adding Check for Dataset API", log.Error(err))
	}

	if err = hc.AddCheck("Vault", vc.Checker); err != nil {
		log.Event(nil, "Error Adding Check for Vault", log.Error(err))
	}

	if err = hc.AddCheck("Filter API", fc.Checker); err != nil {
		log.Event(nil, "Error Adding Check for Filter API", log.Error(err))
	}

	if isPublishing {
		if err = hc.AddCheck("Zebedee", zc.Checker); err != nil {
			log.Event(nil, "Error Adding Check for Zebedee", log.Error(err))
		}
	}

	return
}
