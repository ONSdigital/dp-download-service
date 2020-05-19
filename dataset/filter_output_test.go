package dataset

import (
	"errors"
	"testing"

	"github.com/ONSdigital/dp-api-clients-go/filter"
	"github.com/ONSdigital/dp-download-service/dataset/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	testError = errors.New("borked")

	testFilterOutputDownloadParams = Parameters{
		userAuthToken:        "userAuthToken",
		serviceAuthToken:     "serviceAuthToken",
		downloadServiceToken: "downloadServiceToken",
		collectionID:         "collectionID",
		filterOutputID:       "filterOutputID",
	}
)

func TestGetDownloadsForFilterOutput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	Convey("should return the error if filter client get output is unsuccessful", t, func() {
		filterCli := erroringFilterOutputClient(ctrl, testFilterOutputDownloadParams, testError)

		d := Downloader{FilterCli: filterCli}

		downloads, err := d.GetFilterOutputDownloads(nil, testFilterOutputDownloadParams)

		So(downloads.Available, ShouldHaveLength, 0)
		So(downloads.IsPublished, ShouldBeFalse)
		So(err, ShouldResemble, testError)
	})

	Convey("should return publish false if dataset not published", t, func() {
		csvDownload := getTestFilterDownload()
		filterOutput := getTestDatasetFilterOutput(false, &csvDownload)
		filterCli := successfulFilterOutputClient(ctrl, testFilterOutputDownloadParams, filterOutput)

		d := Downloader{FilterCli: filterCli}

		downloads, err := d.GetFilterOutputDownloads(nil, testFilterOutputDownloadParams)

		csv, found := downloads.Available["csv"]
		So(found, ShouldBeTrue)

		So(csv, ShouldResemble, DownloadInfo{
			URL:     csvDownload.URL,
			Size:    csv.Size,
			Public:  csv.Public,
			Private: csv.Private,
			Skipped: csv.Skipped,
		})

		So(downloads.Available, ShouldHaveLength, 1)
		So(downloads.IsPublished, ShouldBeFalse)
		So(err, ShouldBeNil)
	})

	Convey("should return publish false if dataset not published", t, func() {
		csvDownload := getTestFilterDownload()
		filterOutput := getTestDatasetFilterOutput(false, &csvDownload)
		filterCli := successfulFilterOutputClient(ctrl, testFilterOutputDownloadParams, filterOutput)

		d := Downloader{FilterCli: filterCli}

		downloads, err := d.GetFilterOutputDownloads(nil, testFilterOutputDownloadParams)

		So(downloads.Available, ShouldHaveLength, 1)
		csv, found := downloads.Available["csv"]
		So(found, ShouldBeTrue)

		So(csv, ShouldResemble, DownloadInfo{
			URL:     csvDownload.URL,
			Size:    csv.Size,
			Public:  csv.Public,
			Private: csv.Private,
			Skipped: csv.Skipped,
		})

		So(downloads.IsPublished, ShouldBeFalse)
		So(err, ShouldBeNil)
	})

	Convey("should return expected values if downloads is not empty", t, func() {
		csvDownload := getTestFilterDownload()
		filterOutput := getTestDatasetFilterOutput(true, &csvDownload)
		filterCli := successfulFilterOutputClient(ctrl, testFilterOutputDownloadParams, filterOutput)

		d := Downloader{FilterCli: filterCli}

		downloads, err := d.GetFilterOutputDownloads(nil, testFilterOutputDownloadParams)

		So(downloads.Available, ShouldHaveLength, 1)

		csv, found := downloads.Available["csv"]
		So(found, ShouldBeTrue)

		So(csv, ShouldResemble, DownloadInfo{
			URL:     csvDownload.URL,
			Size:    csv.Size,
			Public:  csv.Public,
			Private: csv.Private,
			Skipped: csv.Skipped,
		})

		So(downloads.IsPublished, ShouldBeTrue)
		So(err, ShouldBeNil)
	})

	Convey("should return expected values if downloads is empty", t, func() {
		filterOutput := getTestDatasetFilterOutput(true, nil)
		filterCli := successfulFilterOutputClient(ctrl, testFilterOutputDownloadParams, filterOutput)

		d := Downloader{FilterCli: filterCli}

		downloads, err := d.GetFilterOutputDownloads(nil, testFilterOutputDownloadParams)

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
		gomock.Eq(p.userAuthToken),
		gomock.Eq(p.serviceAuthToken),
		gomock.Eq(p.downloadServiceToken),
		gomock.Eq(p.collectionID),
		gomock.Eq(p.filterOutputID),
	).Times(1).Return(filter.Model{}, err)

	return filterCli
}

func successfulFilterOutputClient(c *gomock.Controller, p Parameters, output filter.Model) *mocks.MockFilterClient {
	filterCli := mocks.NewMockFilterClient(c)

	filterCli.EXPECT().GetOutput(
		nil,
		gomock.Eq(p.userAuthToken),
		gomock.Eq(p.serviceAuthToken),
		gomock.Eq(p.downloadServiceToken),
		gomock.Eq(p.collectionID),
		gomock.Eq(p.filterOutputID),
	).Times(1).Return(output, nil)

	return filterCli
}
