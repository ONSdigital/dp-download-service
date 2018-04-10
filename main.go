package main

import (
	"os"

	"github.com/ONSdigital/go-ns/vault"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/service"
	"github.com/ONSdigital/go-ns/clients/dataset"
	"github.com/ONSdigital/go-ns/clients/filter"
	"github.com/ONSdigital/go-ns/log"
)

func main() {
	log.Namespace = "dp-download-service"

	cfg, err := config.Get()
	if err != nil {
		log.Error(err, nil)
		os.Exit(1)
	}

	log.Info("config on startup", log.Data{"config": cfg})

	dc := dataset.New(cfg.DatasetAPIURL)
	vc, err := vault.CreateVaultClient(cfg.VaultToken, cfg.VaultAddress, 3)
	if err != nil {
		log.ErrorC("could not create a vault client", err, nil)
		os.Exit(1)
	}

	fc := filter.New(cfg.FilterAPIURL)

	region := "eu-west-1"
	sess := session.New(&aws.Config{Region: &region})

	svc := service.Create(
		cfg.BindAddr,
		cfg.SecretKey,
		cfg.DatasetAuthToken,
		cfg.DownloadServiceToken,
		cfg.VaultPath,
		cfg.BucketName,
		cfg.ServiceAuthToken,
		cfg.ZebedeeURL,
		dc,
		fc,
		sess,
		vc,
		cfg.GracefulShutdownTimeout,
		cfg.HealthCheckInterval,
		cfg.IsPublishing,
	)

	svc.Start()
}
