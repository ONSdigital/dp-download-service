package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/service"
	"github.com/ONSdigital/dp-download-service/service/external"
	dpotelgo "github.com/ONSdigital/dp-otel-go"
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

const serviceName = "dp-download-service"

func main() {
	log.Namespace = serviceName
	ctx := context.Background()

	if err := run(ctx); err != nil {
		log.Fatal(ctx, "fatal runtime error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	cfg, err := config.Get()
	if err != nil {
		log.Fatal(ctx, "error getting config", err)
		os.Exit(1)
	}
	log.Info(ctx, "config on startup", log.Data{"config": cfg})

	// Set up Open Telemetry
	otelConfig := dpotelgo.Config{
		OtelServiceName:          cfg.OTServiceName,
		OtelExporterOtlpEndpoint: cfg.OTExporterOTLPEndpoint,
		OtelBatchTimeout:         cfg.OTBatchTimeout,
	}

	otelShutdown, oErr := dpotelgo.SetupOTelSDK(ctx, otelConfig)
	if oErr != nil {
		return fmt.Errorf("error setting up OpenTelemetry - hint: ensure OTEL_EXPORTER_OTLP_ENDPOINT is set. %w", oErr)
	}
	// Handle shutdown properly so nothing leaks.
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	svc, err := service.New(ctx, BuildTime, GitCommit, Version, cfg, &external.External{})
	if err != nil {
		log.Fatal(ctx, "could not set up Download service", err)
		os.Exit(1)
	}

	svc.Run(ctx)

	sig := <-signals // blocks until an os interrupt or a fatal error occurs
	log.Info(ctx, "os signal received", log.Data{"signal": sig})

	return svc.Close(ctx)
}
