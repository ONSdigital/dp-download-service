package main

import (
	"context"
	"os"

	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/service"
	"github.com/ONSdigital/dp-download-service/service/external"
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

	svc, err := service.New(ctx, BuildTime, GitCommit, Version, cfg, &external.External{})
	if err != nil {
		log.Event(ctx, "could not set up Download service", log.FATAL, log.Error(err))
		os.Exit(1)
	}

	svc.Start(ctx)
}
