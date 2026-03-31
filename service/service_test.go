package service_test

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/ONSdigital/dp-authorisation/v2/authorisation"
	authMock "github.com/ONSdigital/dp-authorisation/v2/authorisation/mock"
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
		authConfig := authorisation.Config{
			Enabled:                        true,
			PermissionsAPIURL:              "http://localhost:1112",
			PermissionsCacheUpdateInterval: 5 * time.Second,
			PermissionsMaxCacheTime:        10 * time.Second,
		}

		ctx := context.Background()
		cfg := &config.Config{
			GracefulShutdownTimeout: 5 * time.Minute,
			IsPublishing:            true,
			AuthorisationConfig:     &authConfig,
		}

		mockedDatasetClient := &DatasetClientMock{
			CheckerFunc: checker,
		}

		mockedFilesClient := &FilesClientMock{
			CheckerFunc: checker,
		}

		mockedFilterClient := &FilterClientMock{
			CheckerFunc: checker,
		}

		mockedImageClient := &ImageClientMock{
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

		authmock := &authMock.MiddlewareMock{
			RequireFunc: func(permission string, handlerFunc http.HandlerFunc) http.HandlerFunc {
				return handlerFunc
			},
		}

		mockedDependencies := &DependenciesMock{
			AuthMiddlewareFunc: func(contextMoqParam context.Context, configMoqParam *config.Config) (authorisation.Middleware, error) {
				return authmock, nil
			},
			DatasetClientFunc: func(s string) downloads.DatasetClient {
				return mockedDatasetClient
			},
			FilesClientFunc: func(s string) downloads.FilesClient {
				return mockedFilesClient
			},
			FilterClientFunc: func(s string) downloads.FilterClient {
				return mockedFilterClient
			},
			ImageClientFunc: func(s string) downloads.ImageClient {
				return mockedImageClient
			},
			S3ClientFunc: func(ctx context.Context, cfg *config.Config) (content.S3Client, error) {
				return mockedS3Client, nil
			},
			HealthCheckFunc: func(cfg *config.Config, buildTime, gitCommit, version string) (service.HealthChecker, error) {
				return mockedHealthChecker, nil
			},
			HTTPServerFunc: func(configMoqParam *config.Config, handler http.Handler) service.HTTPServer {
				return mockedHttpServer
			},
		}

		Convey("When all is well", func() {
			svc, err := service.New(ctx, buildTime, gitCommit, version, cfg, mockedDependencies)

			Convey("New should succeed", func() {
				So(svc, ShouldNotBeNil)
				So(err, ShouldBeNil)
				So(svc.GetDatasetClient(), ShouldEqual, mockedDatasetClient)
				So(svc.GetFilesClient(), ShouldEqual, mockedFilesClient)
				So(svc.GetFilterClient(), ShouldEqual, mockedFilterClient)
				So(svc.GetImageClient(), ShouldEqual, mockedImageClient)
				So(svc.GetS3Client(), ShouldEqual, mockedS3Client)
				So(svc.GetShutdownTimeout(), ShouldEqual, cfg.GracefulShutdownTimeout)
				So(svc.GetHealthChecker(), ShouldEqual, mockedHealthChecker)
			})
		})

		// Ensure New fails when any of the client setups fail
		Convey("When S3 setup fails", func() {
			mockedDependencies.S3ClientFunc = func(ctx context.Context, cfg *config.Config) (content.S3Client, error) {
				return nil, errors.New("s3 failure")
			}

			svc, err := service.New(ctx, buildTime, gitCommit, version, cfg, mockedDependencies)

			Convey("New should fail", func() {
				So(svc, ShouldBeNil)
				So(err.Error(), ShouldContainSubstring, "s3 failure")
			})
		})

		Convey("When healthcheck setup fails", func() {
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

		Convey("When files api healthcheck setup fails", func() {
			mockedHealthChecker.AddCheckFunc = func(name string, checker healthcheck.Checker) error {
				return failIfNameMatches(name, "Files API")
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
	})
}
