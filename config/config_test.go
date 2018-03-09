package config

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSpec(t *testing.T) {

	Convey("Given an environment with no environment variables set", t, func() {
		cfg, err := Get()

		Convey("when the config variables are retrieved", func() {
			Convey("there should be no error returned", func() {
				So(err, ShouldBeNil)
			})

			Convey("the values should be set to the expected defaults", func() {
				So(cfg.BindAddr, ShouldEqual, ":23500")
				So(cfg.BucketName, ShouldEqual, "csv-exported")
				So(cfg.DatasetAPIURL, ShouldEqual, "http://localhost:22000")
				So(cfg.DatasetAuthToken, ShouldEqual, "FD0108EA-825D-411C-9B1D-41EF7727F465")
				So(cfg.XDownloadServiceToken, ShouldEqual, "QB0108EZ-825D-412C-9B1D-41EF7747F462")
				So(cfg.GracefulShutdownTimeout, ShouldEqual, 5*time.Second)
				So(cfg.HealthCheckInterval, ShouldEqual, 1*time.Minute)
				So(cfg.SecretKey, ShouldEqual, "AL0108EA-825D-411C-9B1D-41EF7727F465")
				So(cfg.VaultToken, ShouldEqual, "")
				So(cfg.VaultPath, ShouldEqual, "secret/shared/psk")
			})
		})

	})
}
