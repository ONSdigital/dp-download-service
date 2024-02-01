package config

import (
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/kelseyhightower/envconfig"

	. "github.com/smartystreets/goconvey/convey"
)

// gets the relevant environmental variables for this config and returns them in a map
func getConfigEnv() map[string]string {
	return map[string]string{
		"BIND_ADDR":                    os.Getenv("BIND_ADDR"),
		"BUCKET_NAME":                  os.Getenv("BUCKET_NAME"),
		"DATASET_API_URL":              os.Getenv("DATASET_API_URL"),
		"DOWNLOAD_SERVICE_TOKEN":       os.Getenv("DOWNLOAD_SERVICE_TOKEN"),
		"DATASET_AUTH_TOKEN":           os.Getenv("DATASET_AUTH_TOKEN"),
		"FILTER_API_URL":               os.Getenv("FILTER_API_URL"),
		"IMAGE_API_URL":                os.Getenv("IMAGE_API_URL"),
		"GRACEFUL_SHUTDOWN_TIMEOUT":    os.Getenv("GRACEFUL_SHUTDOWN_TIMEOUT"),
		"HEALTHCHECK_INTERVAL":         os.Getenv("HEALTHCHECK_INTERVAL"),
		"HEALTHCHECK_CRITICAL_TIMEOUT": os.Getenv("HEALTHCHECK_CRITICAL_TIMEOUT"),
		"SERVICE_AUTH_TOKEN":           os.Getenv("SERVICE_AUTH_TOKEN"),
		"SECRET_KEY":                   os.Getenv("SECRET_KEY"),
		"ZEBEDEE_URL":                  os.Getenv("ZEBEDEE_URL"),
		"IS_PUBLISHING":                os.Getenv("IS_PUBLISHING"),
		"ENABLE_MONGO":                 os.Getenv("ENABLE_MONGO"),
		"MONGODB_BIND_ADDR":            os.Getenv("MONGODB_BIND_ADDR"),
		"MONGODB_COLLECTION":           os.Getenv("MONGODB_COLLECTION"),
		"MONGODB_DATABASE":             os.Getenv("MONGODB_DATABASE"),
		"MONGODB_USERNAME":             os.Getenv("MONGODB_USERNAME"),
		"MONGODB_PASSWORD":             os.Getenv("MONGODB_PASSWORD"),
		"MONGODB_IS_SSL":               os.Getenv("MONGODB_IS_SSL"),
		"PUBLIC_BUCKET_URL":            os.Getenv("PUBLIC_BUCKET_URL"),
		"MAX_CONCURRENT_HANDLERS":      os.Getenv("MAX_CONCURRENT_HANDLERS"),
	}
}

func setConfigEnv(configEnv map[string]string) {
	for k, v := range configEnv {
		os.Setenv(k, v)
	}
}

func TestSpec(t *testing.T) {

	Convey("Given an environment with no environment variables set", t, func() {
		originalConfigEnv := getConfigEnv()
		defer setConfigEnv(originalConfigEnv)

		for k := range originalConfigEnv {
			os.Unsetenv(k)
		}

		os.Setenv("PUBLIC_BUCKET_URL", "http://test")

		config, err := Get()

		Convey("when the config variables are retrieved", func() {

			Convey("there should be no error returned", func() {
				So(err, ShouldBeNil)
			})

			Convey("the values should be set to the expected defaults", func() {
				So(config.BindAddr, ShouldEqual, "localhost:23600")
				So(config.BucketName, ShouldEqual, "csv-exported")
				So(config.DatasetAPIURL, ShouldEqual, "http://localhost:22000")
				So(config.DatasetAuthToken, ShouldEqual, "FD0108EA-825D-411C-9B1D-41EF7727F465")
				So(config.DownloadServiceToken, ShouldEqual, "QB0108EZ-825D-412C-9B1D-41EF7747F462")
				So(config.FilterAPIURL, ShouldEqual, "http://localhost:22100")
				So(config.ImageAPIURL, ShouldEqual, "http://localhost:24700")
				So(config.GracefulShutdownTimeout, ShouldEqual, 5*time.Second)
				So(config.HealthCheckInterval, ShouldEqual, 30*time.Second)
				So(config.HealthCheckCriticalTimeout, ShouldEqual, 90*time.Second)
				So(config.ServiceAuthToken, ShouldEqual, "c60198e9-1864-4b68-ad0b-1e858e5b46a4")
				So(config.ZebedeeURL, ShouldEqual, "http://localhost:8082")
				So(config.LocalObjectStore, ShouldEqual, "")
				So(config.MinioAccessKey, ShouldEqual, "")
				So(config.MinioSecretKey, ShouldEqual, "")
				So(config.IsPublishing, ShouldBeTrue)
				So(config.MaxConcurrentHandlers, ShouldEqual, 0)

				expectedUrl, _ := url.Parse("http://test")
				So(config.PublicBucketURL, ShouldResemble, ConfigUrl{*expectedUrl})
			})
		})
	})
}

func TestBadPublicBucketUrl(t *testing.T) {
	Convey("Given an environment variable with a bad public-bucket url", t, func() {
		originalConfigEnv := getConfigEnv()
		defer setConfigEnv(originalConfigEnv)

		for k := range originalConfigEnv {
			os.Unsetenv(k)
		}

		os.Setenv("PUBLIC_BUCKET_URL", "://test")

		cfg = nil

		_, err := Get()

		Convey("getting config values should result in parse error", func() {
			So(err, ShouldHaveSameTypeAs, &envconfig.ParseError{})
		})
	})
}
