package main

import (
	"context"
	"os"

	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/service"
	"github.com/ONSdigital/dp-download-service/service/external"
	"github.com/ONSdigital/log.go/v2/log"
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
		log.Fatal(ctx, "error getting config", err)
		os.Exit(1)
	}
	log.Info(ctx, "config on startup", log.Data{"config": cfg})

	svc, err := service.New(ctx, BuildTime, GitCommit, Version, cfg, &external.External{})
	if err != nil {
		log.Fatal(ctx, "could not set up Download service", err)
		os.Exit(1)
	}

	svc.Start(ctx)
}
