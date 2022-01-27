package service_test

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/content"
	"github.com/ONSdigital/dp-download-service/downloads"
	"github.com/ONSdigital/dp-download-service/service"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNew(t *testing.T) {
	buildTime := "buildTime"
	gitCommit := "gitCommit"
	version := "version"

	buf := bytes.NewBufferString("")
	log.SetOutput(buf)

	// We are not testing the checker function or its return value; we only need
	// a valid function to attach to clients.
	checker := func(ctx context.Context, check *healthcheck.CheckState) error {
		return nil
	}

	Convey("Setting up dependencies", t, func() {

		// Set up happy path clients and dependencies.
		//

		ctx := context.Background()
		cfg := &config.Config{
			GracefulShutdownTimeout: 5 * time.Minute,
			IsPublishing:            true,
			EncryptionDisabled:      false, // just to be explicit
		}

		mockedDatasetClient := &DatasetClientMock{
			CheckerFunc: checker,
		}

		mockedFilterClient := &FilterClientMock{
			CheckerFunc: checker,
		}

		mockedImageClient := &ImageClientMock{
			CheckerFunc: checker,
		}

		mockedVaultClient := &VaultClientMock{
			CheckerFunc: checker,
		}

		mockedS3Client := &S3ClientMock{
			CheckerFunc: checker,
		}

		mockedHealthChecker := &HealthCheckerMock{
			AddCheckFunc: func(s string, checker healthcheck.Checker) error {
				return nil
			},
		}

		mockedHttpServer := &HTTPServerMock{}

		mockedDependencies := &DependenciesMock{
			DatasetClientFunc: func(s string) downloads.DatasetClient {
				return mockedDatasetClient
			},
			FilterClientFunc: func(s string) downloads.FilterClient {
				return mockedFilterClient
			},
			ImageClientFunc: func(s string) downloads.ImageClient {
				return mockedImageClient
			},
			VaultClientFunc: func(cfg *config.Config) (content.VaultClient, error) {
				return mockedVaultClient, nil
			},
			S3ClientFunc: func(cfg *config.Config) (content.S3Client, error) {
				return mockedS3Client, nil
			},
			HealthCheckFunc: func(cfg *config.Config, buildTime, gitCommit, version string) (service.HealthChecker, error) {
				return mockedHealthChecker, nil
			},
			HttpServerFunc: func(configMoqParam *config.Config, handler http.Handler) service.HTTPServer {
				return mockedHttpServer
			},
		}

		Convey("When all is well", func() {
			svc, err := service.New(ctx, buildTime, gitCommit, version, cfg, mockedDependencies)

			Convey("New should succeed", func() {
				So(svc, ShouldNotBeNil)
				So(err, ShouldBeNil)
				So(svc.GetDatasetClient(), ShouldEqual, mockedDatasetClient)
				So(svc.GetFilterClient(), ShouldEqual, mockedFilterClient)
				So(svc.GetImageClient(), ShouldEqual, mockedImageClient)
				So(svc.GetVaultClient(), ShouldEqual, mockedVaultClient)
				So(svc.GetS3Client(), ShouldEqual, mockedS3Client)
				So(svc.GetZebedeeHealthClient(), ShouldNotBeNil)
				So(svc.GetShutdownTimeout(), ShouldEqual, cfg.GracefulShutdownTimeout)
				So(svc.GetHealthChecker(), ShouldEqual, mockedHealthChecker)
			})
		})

		// Ensure New fails when any of the client setups fail
		//

		Convey("When Vault setup fails", func() {
			mockedDependencies.VaultClientFunc = func(cfg *config.Config) (content.VaultClient, error) {
				return nil, errors.New("vault failure")
			}

			svc, err := service.New(ctx, buildTime, gitCommit, version, cfg, mockedDependencies)

			Convey("New should fail", func() {
				So(svc, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "vault failure")
			})
		})

		Convey("When S3 setup fails", func() {
			mockedDependencies.S3ClientFunc = func(cfg *config.Config) (content.S3Client, error) {
				return nil, errors.New("s3 failure")
			}

			svc, err := service.New(ctx, buildTime, gitCommit, version, cfg, mockedDependencies)

			Convey("New should fail", func() {
				So(svc, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "s3 failure")
			})
		})

		Convey("When healthcheck setup fail", func() {
			mockedDependencies.HealthCheckFunc = func(cfg *config.Config, buildTime, gitCommit, version string) (service.HealthChecker, error) {
				return nil, errors.New("healthcheck failure")
			}

			svc, err := service.New(ctx, buildTime, gitCommit, version, cfg, mockedDependencies)

			Convey("New should fail", func() {
				So(svc, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "healthcheck failure")
			})
		})

		// Ensure New fails if any of the healthcheck AddChecks fail
		//

		failIfNameMatches := func(name, match string) error {
			if name == match {
				return errors.New(name)
			}
			return nil
		}

		Convey("When dataset api healthcheck setup fails", func() {
			mockedHealthChecker.AddCheckFunc = func(name string, checker healthcheck.Checker) error {
				return failIfNameMatches(name, "Dataset API")
			}

			svc, err := service.New(ctx, buildTime, gitCommit, version, cfg, mockedDependencies)

			Convey("New should fail", func() {
				So(svc, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "registering checkers for healthcheck")
			})
		})

		Convey("When vault healthcheck setup fails", func() {
			mockedHealthChecker.AddCheckFunc = func(name string, checker healthcheck.Checker) error {
				return failIfNameMatches(name, "Vault")
			}

			svc, err := service.New(ctx, buildTime, gitCommit, version, cfg, mockedDependencies)

			Convey("New should fail", func() {
				So(svc, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "registering checkers for healthcheck")
			})
		})

		Convey("When filter api healthcheck setup fails", func() {
			mockedHealthChecker.AddCheckFunc = func(name string, checker healthcheck.Checker) error {
				return failIfNameMatches(name, "Filter API")
			}

			svc, err := service.New(ctx, buildTime, gitCommit, version, cfg, mockedDependencies)

			Convey("New should fail", func() {
				So(svc, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "registering checkers for healthcheck")
			})
		})

		Convey("When image api healthcheck setup fails", func() {
			mockedHealthChecker.AddCheckFunc = func(name string, checker healthcheck.Checker) error {
				return failIfNameMatches(name, "Image API")
			}

			svc, err := service.New(ctx, buildTime, gitCommit, version, cfg, mockedDependencies)

			Convey("New should fail", func() {
				So(svc, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "registering checkers for healthcheck")
			})
		})

		Convey("When S3 healthcheck setup fails", func() {
			mockedHealthChecker.AddCheckFunc = func(name string, checker healthcheck.Checker) error {
				return failIfNameMatches(name, "S3")
			}

			svc, err := service.New(ctx, buildTime, gitCommit, version, cfg, mockedDependencies)

			Convey("New should fail", func() {
				So(svc, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "registering checkers for healthcheck")
			})
		})

		Convey("When zebedee healthcheck setup fails", func() {
			mockedHealthChecker.AddCheckFunc = func(name string, checker healthcheck.Checker) error {
				return failIfNameMatches(name, "Zebedee")
			}

			svc, err := service.New(ctx, buildTime, gitCommit, version, cfg, mockedDependencies)

			Convey("New should fail", func() {
				So(svc, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "registering checkers for healthcheck")
			})
		})

		// Some feature flag tests
		//

		// Ensure Vault client not created if encryption is disabled
		//
		Convey("When encryption is disabled", func() {
			// fail New() if vault client setup is called
			mockedDependencies.VaultClientFunc = func(cfg *config.Config) (content.VaultClient, error) {
				return nil, errors.New("vault failure")
			}
			// fail New() if vault checker added
			mockedHealthChecker.AddCheckFunc = func(name string, checker healthcheck.Checker) error {
				return failIfNameMatches(name, "Vault")
			}

			cfg.EncryptionDisabled = true
			svc, err := service.New(ctx, buildTime, gitCommit, version, cfg, mockedDependencies)

			Convey("Vault should not be setup", func() {
				So(svc, ShouldNotBeNil)
				So(err, ShouldBeNil)
				So(svc.GetVaultClient(), ShouldBeNil)
			})
		})

		// Ensure Zebedee health check setup is not run when IsPublish is false
		//
		Convey("When IsPublishing is false", func() {
			mockedHealthChecker.AddCheckFunc = func(name string, checker healthcheck.Checker) error {
				return failIfNameMatches(name, "Zebedee")
			}

			cfg.IsPublishing = false // just to be explicit
			svc, err := service.New(ctx, buildTime, gitCommit, version, cfg, mockedDependencies)

			Convey("Zebedee should not be set up", func() {
				So(svc, ShouldNotBeNil)
				So(err, ShouldBeNil)
				So(svc.GetZebedeeHealthClient(), ShouldBeNil)
			})
		})
	})
}
