package downloads

import (
	"testing"

	"github.com/ONSdigital/dp-api-clients-go/dataset"
	"github.com/ONSdigital/dp-download-service/downloads/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	testDatasetVersionDownloadParams = Parameters{
		UserAuthToken:        "userAuthToken",
		ServiceAuthToken:     "serviceAuthToken",
		DownloadServiceToken: "downloadServiceToken",
		DatasetID:            "datasetID",
		Edition:              "edition",
		Version:              "version",
	}
)

func TestGetDownloadForDataset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	Convey("should return error is dataset client get version returns an error", t, func() {
		datasetCli := erroringDatasetClient(ctrl, testDatasetVersionDownloadParams, testError)
		filterCli := filterOutputClientNeverInvoked(ctrl)

		d := Downloader{
			DatasetCli: datasetCli,
			FilterCli:  filterCli,
		}

		downloads, err := d.Get(nil, testDatasetVersionDownloadParams)

		So(downloads.Available, ShouldHaveLength, 0)
		So(downloads.IsPublished, ShouldBeFalse)
		So(err, ShouldResemble, testError)
	})

	Convey("should return published false if dataset state not published", t, func() {
		datasetDownload := testDatasetDownload()
		datasetVersion := testDatasetVersion("not published", &datasetDownload)

		datasetCli := successfulDatasetClient(ctrl, testDatasetVersionDownloadParams, datasetVersion)
		filterCli := filterOutputClientNeverInvoked(ctrl)

		d := Downloader{
			DatasetCli: datasetCli,
			FilterCli:  filterCli,
		}

		downloads, err := d.Get(nil, testDatasetVersionDownloadParams)

		So(downloads.Available, ShouldHaveLength, 1)
		actual, found := downloads.Available["csv"]
		So(found, ShouldBeTrue)

		So(actual.Skipped, ShouldBeFalse)
		So(actual.URL, ShouldEqual, datasetDownload.URL)
		So(actual.Private, ShouldEqual, datasetDownload.Private)
		So(actual.Public, ShouldEqual, datasetDownload.Public)

		So(downloads.IsPublished, ShouldBeFalse)
		So(err, ShouldBeNil)
	})

	Convey("should return empty downloads if dataset version downloads empty", t, func() {
		datasetVersion := testDatasetVersion("not published", nil)

		datasetCli := successfulDatasetClient(ctrl, testDatasetVersionDownloadParams, datasetVersion)
		filterCli := filterOutputClientNeverInvoked(ctrl)

		d := Downloader{
			DatasetCli: datasetCli,
			FilterCli:  filterCli,
		}

		downloads, err := d.Get(nil, testDatasetVersionDownloadParams)

		So(downloads.Available, ShouldHaveLength, 0)
		So(downloads.IsPublished, ShouldBeFalse)
		So(err, ShouldBeNil)
	})

	Convey("should return downloads if dataset version downloads not empty", t, func() {
		datasetDownload := testDatasetDownload()
		datasetVersion := testDatasetVersion("not published", &datasetDownload)

		datasetCli := successfulDatasetClient(ctrl, testDatasetVersionDownloadParams, datasetVersion)
		filterCli := filterOutputClientNeverInvoked(ctrl)

		d := Downloader{
			DatasetCli: datasetCli,
			FilterCli:  filterCli,
		}

		downloads, err := d.Get(nil, testDatasetVersionDownloadParams)

		So(downloads.Available, ShouldHaveLength, 1)
		actual, found := downloads.Available["csv"]
		So(found, ShouldBeTrue)
		So(actual.Skipped, ShouldBeFalse)
		So(actual.URL, ShouldEqual, datasetDownload.URL)
		So(actual.Private, ShouldEqual, datasetDownload.Private)
		So(actual.Public, ShouldEqual, datasetDownload.Public)

		So(downloads.IsPublished, ShouldBeFalse)
		So(err, ShouldBeNil)
	})
}

func erroringDatasetClient(c *gomock.Controller, p Parameters, err error) *mocks.MockDatasetClient {
	cli := mocks.NewMockDatasetClient(c)

	cli.EXPECT().GetVersion(
		gomock.Any(),
		gomock.Eq(p.UserAuthToken),
		gomock.Eq(p.ServiceAuthToken),
		gomock.Eq(p.DownloadServiceToken),
		gomock.Eq(p.CollectionID),
		gomock.Eq(p.DatasetID),
		gomock.Eq(p.Edition),
		gomock.Eq(p.Version),
	).Times(1).Return(dataset.Version{}, err)
	return cli
}

func testDatasetDownload() dataset.Download {
	return dataset.Download{
		URL:     "/abc",
		Size:    "1",
		Public:  "/public",
		Private: "/private",
	}
}

func testDatasetVersion(state string, dl *dataset.Download) dataset.Version {
	version := dataset.Version{
		State: state,
	}

	if dl != nil {
		version.Downloads = map[string]dataset.Download{"csv": *dl}
	}

	return version
}

func successfulDatasetClient(c *gomock.Controller, p Parameters, v dataset.Version) *mocks.MockDatasetClient {
	cli := mocks.NewMockDatasetClient(c)

	cli.EXPECT().GetVersion(
		gomock.Any(),
		gomock.Eq(p.UserAuthToken),
		gomock.Eq(p.ServiceAuthToken),
		gomock.Eq(p.DownloadServiceToken),
		gomock.Eq(p.CollectionID),
		gomock.Eq(p.DatasetID),
		gomock.Eq(p.Edition),
		gomock.Eq(p.Version),
	).Times(1).Return(v, nil)
	return cli
}
func datasetClientNeverInvoked(c *gomock.Controller) *mocks.MockDatasetClient {
	cli := mocks.NewMockDatasetClient(c)

	cli.EXPECT().GetVersion(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Times(0).Return(dataset.Version{}, nil)
	return cli
}
