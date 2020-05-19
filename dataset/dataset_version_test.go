package dataset

import (
	"testing"

	"github.com/ONSdigital/dp-api-clients-go/dataset"
	"github.com/ONSdigital/dp-download-service/handlers/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	testDatasetVersionDownloadParams = Parameters{
		userAuthToken:        "userAuthToken",
		serviceAuthToken:     "serviceAuthToken",
		downloadServiceToken: "downloadServiceToken",
		datasetID:            "datasetID",
		edition:              "edition",
		version:              "version",
	}
)

func TestGetDownloadForDataset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	Convey("should return error is dataset client get version returns an error", t, func() {
		datasetCli := erroringDatasetClient(ctrl, testDatasetVersionDownloadParams, testError)

		d := Downloader{DatasetCli: datasetCli}

		downloads, err := d.GetDatasetVersionDownloads(nil, testDatasetVersionDownloadParams)

		So(downloads.Available, ShouldHaveLength, 0)
		So(downloads.IsPublished, ShouldBeFalse)
		So(err, ShouldResemble, testError)
	})

	Convey("should return published false if dataset state not published", t, func() {
		datasetDownload := testDatasetDownload()
		datasetVersion := testDatasetVersion("not published", &datasetDownload)
		datasetCli := successfulDatasetClient(ctrl, testDatasetVersionDownloadParams, datasetVersion)

		d := Downloader{DatasetCli: datasetCli}

		downloads, err := d.GetDatasetVersionDownloads(nil, testDatasetVersionDownloadParams)

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

		d := Downloader{DatasetCli: datasetCli}

		downloads, err := d.GetDatasetVersionDownloads(nil, testDatasetVersionDownloadParams)

		So(downloads.Available, ShouldHaveLength, 0)
		So(downloads.IsPublished, ShouldBeFalse)
		So(err, ShouldBeNil)
	})

	Convey("should return downloads if dataset version downloads not empty", t, func() {
		datasetDownload := testDatasetDownload()
		datasetVersion := testDatasetVersion("not published", &datasetDownload)
		datasetCli := successfulDatasetClient(ctrl, testDatasetVersionDownloadParams, datasetVersion)

		d := Downloader{DatasetCli: datasetCli}

		downloads, err := d.GetDatasetVersionDownloads(nil, testDatasetVersionDownloadParams)

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
		gomock.Eq(p.userAuthToken),
		gomock.Eq(p.serviceAuthToken),
		gomock.Eq(p.downloadServiceToken),
		gomock.Eq(p.collectionID),
		gomock.Eq(p.datasetID),
		gomock.Eq(p.edition),
		gomock.Eq(p.version),
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
		gomock.Eq(p.userAuthToken),
		gomock.Eq(p.serviceAuthToken),
		gomock.Eq(p.downloadServiceToken),
		gomock.Eq(p.collectionID),
		gomock.Eq(p.datasetID),
		gomock.Eq(p.edition),
		gomock.Eq(p.version),
	).Times(1).Return(v, nil)
	return cli
}
