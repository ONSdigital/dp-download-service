package downloads

import (
	"errors"
	"testing"

	"github.com/ONSdigital/dp-api-clients-go/dataset"
	"github.com/ONSdigital/dp-download-service/downloads/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	testCSVPublicUrl        = "http://public.localhost/public/filename.csv"
	testCSVPrivateUrl       = "http://private.localhost/private/filename.csv"
	testCSVPrivateFilename  = "filename.csv"
	testCSVPrivateS3Path    = "private/filename.csv"
	testCSVPrivateVaultPath = "filename.csv"
	testBadPrivateURL       = "@£$%^&*()_+"
)

var (
	testErrDataset                   = errors.New("borked dataset")
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
		datasetCli := erroringDatasetClient(ctrl, testDatasetVersionDownloadParams, testErrDataset)
		filterCli := filterOutputClientNeverInvoked(ctrl)
		imgCli := imageClientNeverInvoked(ctrl)

		d := Downloader{
			DatasetCli: datasetCli,
			FilterCli:  filterCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(nil, testDatasetVersionDownloadParams, TypeDatasetVersion, "")

		So(downloads.IsPublished, ShouldBeFalse)
		So(downloads.Public, ShouldBeBlank)
		So(downloads.PrivateFilename, ShouldBeBlank)
		So(downloads.PrivateS3Path, ShouldBeBlank)
		So(downloads.PrivateVaultPath, ShouldBeBlank)
		So(err, ShouldResemble, testErrDataset)
	})

	Convey("should return error if privateURL is invalid", t, func() {
		datasetDownload := testDatasetDownloadBadURL()
		datasetVersion := testDatasetVersion("not published", &datasetDownload)

		datasetCli := successfulDatasetClient(ctrl, testDatasetVersionDownloadParams, datasetVersion)
		filterCli := filterOutputClientNeverInvoked(ctrl)
		imgCli := imageClientNeverInvoked(ctrl)

		d := Downloader{
			DatasetCli: datasetCli,
			FilterCli:  filterCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(nil, testDatasetVersionDownloadParams, TypeDatasetVersion, "csv")

		So(downloads.IsPublished, ShouldBeFalse)
		So(downloads.Public, ShouldBeBlank)
		So(downloads.PrivateFilename, ShouldBeBlank)
		So(downloads.PrivateS3Path, ShouldBeBlank)
		So(downloads.PrivateVaultPath, ShouldBeBlank)
		So(err, ShouldNotBeNil)
	})

	Convey("should return published false if dataset state not published", t, func() {
		datasetDownload := testDatasetDownload()
		datasetVersion := testDatasetVersion("not published", &datasetDownload)

		datasetCli := successfulDatasetClient(ctrl, testDatasetVersionDownloadParams, datasetVersion)
		filterCli := filterOutputClientNeverInvoked(ctrl)
		imgCli := imageClientNeverInvoked(ctrl)

		d := Downloader{
			DatasetCli: datasetCli,
			FilterCli:  filterCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(nil, testDatasetVersionDownloadParams, TypeDatasetVersion, "csv")

		So(downloads.IsPublished, ShouldBeFalse)
		So(downloads.Public, ShouldResemble, testCSVPublicUrl)
		So(downloads.PrivateFilename, ShouldResemble, testCSVPrivateFilename)
		So(downloads.PrivateS3Path, ShouldResemble, testCSVPrivateS3Path)
		So(downloads.PrivateVaultPath, ShouldResemble, testCSVPrivateVaultPath)
		So(err, ShouldBeNil)
	})

	Convey("should return empty downloads if dataset version downloads empty", t, func() {
		datasetVersion := testDatasetVersion("not published", nil)

		datasetCli := successfulDatasetClient(ctrl, testDatasetVersionDownloadParams, datasetVersion)
		filterCli := filterOutputClientNeverInvoked(ctrl)
		imgCli := imageClientNeverInvoked(ctrl)

		d := Downloader{
			DatasetCli: datasetCli,
			FilterCli:  filterCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(nil, testDatasetVersionDownloadParams, TypeDatasetVersion, "csv")

		So(downloads.IsPublished, ShouldBeFalse)
		So(downloads.Public, ShouldBeBlank)
		So(downloads.PrivateFilename, ShouldBeBlank)
		So(downloads.PrivateS3Path, ShouldBeBlank)
		So(downloads.PrivateVaultPath, ShouldBeBlank)
		So(err, ShouldBeNil)
	})

	Convey("should return downloads if dataset version downloads not empty", t, func() {
		datasetDownload := testDatasetDownload()
		datasetVersion := testDatasetVersion("not published", &datasetDownload)

		datasetCli := successfulDatasetClient(ctrl, testDatasetVersionDownloadParams, datasetVersion)
		filterCli := filterOutputClientNeverInvoked(ctrl)
		imgCli := imageClientNeverInvoked(ctrl)

		d := Downloader{
			DatasetCli: datasetCli,
			FilterCli:  filterCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(nil, testDatasetVersionDownloadParams, TypeDatasetVersion, "csv")

		So(downloads.IsPublished, ShouldBeFalse)
		So(downloads.Public, ShouldResemble, testCSVPublicUrl)
		So(downloads.PrivateFilename, ShouldResemble, testCSVPrivateFilename)
		So(downloads.PrivateS3Path, ShouldResemble, testCSVPrivateS3Path)
		So(downloads.PrivateVaultPath, ShouldResemble, testCSVPrivateVaultPath)
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
		Public:  testCSVPublicUrl,
		Private: testCSVPrivateUrl,
	}
}

func testDatasetDownloadBadURL() dataset.Download {
	return dataset.Download{
		URL:     "/abc",
		Size:    "1",
		Public:  testCSVPublicUrl,
		Private: testBadPrivateURL,
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
