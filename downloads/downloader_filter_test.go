package downloads

import (
	"context"
	"errors"
	"testing"

	"github.com/ONSdigital/dp-api-clients-go/v2/filter"
	"github.com/ONSdigital/dp-download-service/downloads/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	errFilter = errors.New("borked filter")

	testFilterOutputDownloadParams = Parameters{
		UserAuthToken:        "userAuthToken",
		ServiceAuthToken:     "serviceAuthToken",
		DownloadServiceToken: "downloadServiceToken",
		CollectionID:         "collectionID",
		FilterOutputID:       "filterOutputID",
	}
)

func TestGetDownloadsForFilterOutput(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	Convey("should return the error if filter client get output is unsuccessful", t, func() {
		filterCli := erroringFilterOutputClient(ctrl, testFilterOutputDownloadParams, errFilter)
		datasetCli := datasetClientNeverInvoked(ctrl)
		imgCli := imageClientNeverInvoked(ctrl)

		d := Downloader{
			DatasetCli: datasetCli,
			FilterCli:  filterCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(ctx, testFilterOutputDownloadParams, TypeFilterOutput, "csv")

		So(downloads.IsPublished, ShouldBeFalse)
		So(downloads.Public, ShouldBeBlank)
		So(downloads.PrivateFilename, ShouldBeBlank)
		So(downloads.PrivateS3Path, ShouldBeBlank)
		So(err, ShouldResemble, errFilter)
	})

	Convey("should return error if privateURL is invalid", t, func() {
		csvDownload := getTestFilterDownloadBadURL()
		filterOutput := getTestDatasetFilterOutput(false, &csvDownload)

		filterCli := successfulFilterOutputClient(ctrl, testFilterOutputDownloadParams, filterOutput)
		datasetCli := datasetClientNeverInvoked(ctrl)
		imgCli := imageClientNeverInvoked(ctrl)

		d := Downloader{
			DatasetCli: datasetCli,
			FilterCli:  filterCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(ctx, testFilterOutputDownloadParams, TypeFilterOutput, "csv")

		So(downloads.IsPublished, ShouldBeFalse)
		So(downloads.Public, ShouldBeBlank)
		So(downloads.PrivateFilename, ShouldBeBlank)
		So(downloads.PrivateS3Path, ShouldBeBlank)
		So(err, ShouldNotBeNil)
	})

	Convey("should return publish false if dataset not published", t, func() {
		csvDownload := getTestFilterDownload()
		filterOutput := getTestDatasetFilterOutput(false, &csvDownload)

		filterCli := successfulFilterOutputClient(ctrl, testFilterOutputDownloadParams, filterOutput)
		datasetCli := datasetClientNeverInvoked(ctrl)
		imgCli := imageClientNeverInvoked(ctrl)

		d := Downloader{
			DatasetCli: datasetCli,
			FilterCli:  filterCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(ctx, testFilterOutputDownloadParams, TypeFilterOutput, "csv")

		So(downloads.IsPublished, ShouldBeFalse)
		So(downloads.Public, ShouldResemble, testCSVPublicUrl)
		So(downloads.PrivateFilename, ShouldResemble, testCSVPrivateFilename)
		So(downloads.PrivateS3Path, ShouldResemble, testCSVPrivateS3Path)
		So(err, ShouldBeNil)
	})

	Convey("should return expected values if downloads is not empty", t, func() {
		csvDownload := getTestFilterDownload()
		filterOutput := getTestDatasetFilterOutput(true, &csvDownload)

		filterCli := successfulFilterOutputClient(ctrl, testFilterOutputDownloadParams, filterOutput)
		datasetCli := datasetClientNeverInvoked(ctrl)
		imgCli := imageClientNeverInvoked(ctrl)

		d := Downloader{
			DatasetCli: datasetCli,
			FilterCli:  filterCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(ctx, testFilterOutputDownloadParams, TypeFilterOutput, "csv")

		So(downloads.IsPublished, ShouldBeTrue)
		So(downloads.Public, ShouldResemble, testCSVPublicUrl)
		So(downloads.PrivateFilename, ShouldResemble, testCSVPrivateFilename)
		So(downloads.PrivateS3Path, ShouldResemble, testCSVPrivateS3Path)
		So(err, ShouldBeNil)
	})

	Convey("should return expected values if downloads is empty", t, func() {
		filterOutput := getTestDatasetFilterOutput(true, nil)

		filterCli := successfulFilterOutputClient(ctrl, testFilterOutputDownloadParams, filterOutput)
		datasetCli := datasetClientNeverInvoked(ctrl)
		imgCli := imageClientNeverInvoked(ctrl)

		d := Downloader{
			DatasetCli: datasetCli,
			FilterCli:  filterCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(ctx, testFilterOutputDownloadParams, TypeFilterOutput, "csv")

		So(downloads.IsPublished, ShouldBeTrue)
		So(downloads.Public, ShouldBeBlank)
		So(downloads.PrivateFilename, ShouldBeBlank)
		So(downloads.PrivateS3Path, ShouldBeBlank)
		So(err, ShouldBeNil)
	})
}

func getTestFilterDownload() filter.Download {
	return filter.Download{
		URL:     "/downloadURL",
		Size:    "666",
		Public:  testCSVPublicUrl,
		Private: testCSVPrivateUrl,
		Skipped: false,
	}
}

func getTestFilterDownloadBadURL() filter.Download {
	return filter.Download{
		URL:     "/downloadURL",
		Size:    "666",
		Public:  testCSVPublicUrl,
		Private: testBadPrivateURL,
		Skipped: false,
	}
}

func getTestDatasetFilterOutput(isPublished bool, dl *filter.Download) filter.Model {
	f := filter.Model{IsPublished: isPublished}

	if dl != nil {
		f.Downloads = map[string]filter.Download{
			"csv": *dl,
		}
	}
	return f
}

func erroringFilterOutputClient(c *gomock.Controller, p Parameters, err error) *mocks.MockFilterClient {
	filterCli := mocks.NewMockFilterClient(c)

	filterCli.EXPECT().GetOutput(
		gomock.Any(),
		gomock.Eq(p.UserAuthToken),
		gomock.Eq(p.ServiceAuthToken),
		gomock.Eq(p.DownloadServiceToken),
		gomock.Eq(p.CollectionID),
		gomock.Eq(p.FilterOutputID),
	).Times(1).Return(filter.Model{}, err)

	return filterCli
}

func successfulFilterOutputClient(c *gomock.Controller, p Parameters, output filter.Model) *mocks.MockFilterClient {
	filterCli := mocks.NewMockFilterClient(c)

	filterCli.EXPECT().GetOutput(
		gomock.Any(),
		gomock.Eq(p.UserAuthToken),
		gomock.Eq(p.ServiceAuthToken),
		gomock.Eq(p.DownloadServiceToken),
		gomock.Eq(p.CollectionID),
		gomock.Eq(p.FilterOutputID),
	).Times(1).Return(output, nil)

	return filterCli
}

func filterOutputClientNeverInvoked(c *gomock.Controller) *mocks.MockFilterClient {
	filterCli := mocks.NewMockFilterClient(c)

	filterCli.EXPECT().GetOutput(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Times(0).Return(filter.Model{}, nil)

	return filterCli
}
