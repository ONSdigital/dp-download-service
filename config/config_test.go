package config

import (
	"net/url"
	"os"
	"testing"
	"time"

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
		"VAULT_TOKEN":                  os.Getenv("VAULT_TOKEN"),
		"VAULT_ADDR":                   os.Getenv("VAULT_ADDR"),
		"VAULT_PATH":                   os.Getenv("VAULT_PATH"),
		"ZEBEDEE_URL":                  os.Getenv("ZEBEDEE_URL"),
		"IS_PUBLISHING":                os.Getenv("IS_PUBLISHING"),
		"ENCRYPTION_DISABLED":          os.Getenv("ENCRYPTION_DISABLED"),
		"ENABLE_MONGO":                 os.Getenv("ENABLE_MONGO"),
		"MONGODB_BIND_ADDR":            os.Getenv("MONGODB_BIND_ADDR"),
		"MONGODB_COLLECTION":           os.Getenv("MONGODB_COLLECTION"),
		"MONGODB_DATABASE":             os.Getenv("MONGODB_DATABASE"),
		"MONGODB_USERNAME":             os.Getenv("MONGODB_USERNAME"),
		"MONGODB_PASSWORD":             os.Getenv("MONGODB_PASSWORD"),
		"MONGODB_IS_SSL":               os.Getenv("MONGODB_IS_SSL"),
		"PUBLIC_BUCKET_URL":            os.Getenv("PUBLIC_BUCKET_URL"),
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

		cfg, err := Get()

		Convey("when the config variables are retrieved", func() {

			Convey("there should be no error returned", func() {
				So(err, ShouldBeNil)
			})

			Convey("the values should be set to the expected defaults", func() {
				So(cfg.BindAddr, ShouldEqual, "localhost:23600")
				So(cfg.BucketName, ShouldEqual, "csv-exported")
				So(cfg.DatasetAPIURL, ShouldEqual, "http://localhost:22000")
				So(cfg.DatasetAuthToken, ShouldEqual, "FD0108EA-825D-411C-9B1D-41EF7727F465")
				So(cfg.DownloadServiceToken, ShouldEqual, "QB0108EZ-825D-412C-9B1D-41EF7747F462")
				So(cfg.FilterAPIURL, ShouldEqual, "http://localhost:22100")
				So(cfg.ImageAPIURL, ShouldEqual, "http://localhost:24700")
				So(cfg.GracefulShutdownTimeout, ShouldEqual, 5*time.Second)
				So(cfg.HealthCheckInterval, ShouldEqual, 30*time.Second)
				So(cfg.HealthCheckCriticalTimeout, ShouldEqual, 90*time.Second)
				So(cfg.VaultToken, ShouldEqual, "")
				So(cfg.VaultPath, ShouldEqual, "secret/shared/psk")
				So(cfg.ServiceAuthToken, ShouldEqual, "c60198e9-1864-4b68-ad0b-1e858e5b46a4")
				So(cfg.ZebedeeURL, ShouldEqual, "http://localhost:8082")
				So(cfg.LocalObjectStore, ShouldEqual, "")
				So(cfg.MinioAccessKey, ShouldEqual, "")
				So(cfg.MinioSecretKey, ShouldEqual, "")
				So(cfg.IsPublishing, ShouldBeTrue)
				So(cfg.EncryptionDisabled, ShouldBeFalse)

				expectedUrl, _ := url.Parse("http://test")
				So(cfg.PublicBucketURL, ShouldResemble, ConfigUrl{*expectedUrl})
			})
		})
	})
}
