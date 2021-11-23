package main

import (
	"context"
	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/service"
	"github.com/ONSdigital/dp-download-service/service/external"
	"github.com/ONSdigital/log.go/v2/log"
	"os"
	"os/signal"
)

var (
	// BuildTime represents the time in which the service was built
	BuildTime string
	// GitCommit represents the commit (SHA-1) hash of the service that is running
	GitCommit string
	// Version represents the version of the service that is running
	Version string
)

const serviceName = "dp-download-service"

func main()  {
	log.Namespace = serviceName
	ctx := context.Background()

	if err := run(ctx); err != nil {
		log.Fatal(nil, "fatal runtime error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, os.Kill)

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

	svc.Run(ctx)

	// blocks until an os interrupt or a fatal error occurs
	select {
	case sig := <-signals:
		log.Info(ctx, "os signal received", log.Data{"signal": sig})
	}

	return svc.Close(ctx)
}
