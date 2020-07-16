package downloads

import (
	"errors"
	"testing"

	"github.com/ONSdigital/dp-api-clients-go/filter"
	"github.com/ONSdigital/dp-download-service/downloads/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	testErrFilter = errors.New("borked filter")

	testFilterOutputDownloadParams = Parameters{
		UserAuthToken:        "userAuthToken",
		ServiceAuthToken:     "serviceAuthToken",
		DownloadServiceToken: "downloadServiceToken",
		CollectionID:         "collectionID",
		FilterOutputID:       "filterOutputID",
	}
)

func TestGetDownloadsForFilterOutput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	Convey("should return the error if filter client get output is unsuccessful", t, func() {
		filterCli := erroringFilterOutputClient(ctrl, testFilterOutputDownloadParams, testErrFilter)
		datasetCli := datasetClientNeverInvoked(ctrl)
		imgCli := imageClientNeverInvoked(ctrl)

		d := Downloader{
			DatasetCli: datasetCli,
			FilterCli:  filterCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(nil, testFilterOutputDownloadParams, TypeFilterOutput)

		So(downloads.Available, ShouldHaveLength, 0)
		So(downloads.IsPublished, ShouldBeFalse)
		So(err, ShouldResemble, testErrFilter)
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

		downloads, err := d.Get(nil, testFilterOutputDownloadParams, TypeFilterOutput)

		So(downloads.Available, ShouldHaveLength, 1)
		csv, found := downloads.Available["csv"]
		So(found, ShouldBeTrue)

		So(csv, ShouldResemble, Info{
			Public:  csvDownload.Public,
			Private: csvDownload.Private,
		})

		So(downloads.IsPublished, ShouldBeFalse)
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

		downloads, err := d.Get(nil, testFilterOutputDownloadParams, TypeFilterOutput)

		So(downloads.Available, ShouldHaveLength, 1)

		csv, found := downloads.Available["csv"]
		So(found, ShouldBeTrue)

		So(csv, ShouldResemble, Info{
			Public:  csvDownload.Public,
			Private: csvDownload.Private,
		})

		So(downloads.IsPublished, ShouldBeTrue)
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

		downloads, err := d.Get(nil, testFilterOutputDownloadParams, TypeFilterOutput)

		So(downloads.Available, ShouldHaveLength, 0)
		So(downloads.IsPublished, ShouldBeTrue)
		So(err, ShouldBeNil)
	})
}

func getTestFilterDownload() filter.Download {
	return filter.Download{
		URL:     "/downloadURL",
		Size:    "666",
		Public:  "/public/download/url",
		Private: "/private/download/url",
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
		nil,
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
		nil,
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
		nil,
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Times(0).Return(filter.Model{}, nil)

	return filterCli
}
