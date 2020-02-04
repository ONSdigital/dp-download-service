package main

import (
	"context"
	"os"

	"github.com/ONSdigital/go-ns/vault"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/ONSdigital/dp-api-clients-go/dataset"
	"github.com/ONSdigital/dp-api-clients-go/filter"
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
	vc, err := vault.CreateVaultClient(cfg.VaultToken, cfg.VaultAddress, 3)
	if err != nil {
		log.Event(ctx, "could not create a vault client", log.Error(err))
		os.Exit(1)
	}

	fc := filter.New(cfg.FilterAPIURL)

	region := "eu-west-1"
	sess := session.New(&aws.Config{Region: &region})

	svc := service.Create(
		*cfg,
		dc,
		fc,
		sess,
		vc,
		BuildTime,
		GitCommit,
		Version,
	)

	svc.Start()
}
