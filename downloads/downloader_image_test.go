package downloads

import (
	"context"
	"errors"
	"testing"

	"github.com/ONSdigital/dp-api-clients-go/v2/image"
	"github.com/ONSdigital/dp-download-service/downloads/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	testImageID = "myImageID"
	testVariant = "myVariant"
	testName    = "myImageName.png"

	testImagePublicUrl       = "http://public.localhost/images/myImageID/myVariant/myImageName.png"
	testImagePrivateUrl      = "http://private.localhost/images/myImageID/myVariant/myImageName.png"
	testImagePrivateFilename = "myImageName.png"
	testImagePrivateS3Path   = "images/myImageID/myVariant"
)

var (
	errImage = errors.New("borked image")

	testImageDownloadParams = Parameters{
		ImageID:  testImageID,
		Variant:  testVariant,
		Filename: testName,
	}
)

func TestGetDownloadsForImage(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	Convey("should return the error if image client get image is unsuccessful (eg. 404 Not Found)", t, func() {
		imgCli := erroringImageClient(ctrl, testImageDownloadParams, errImage)
		datasetCli := datasetClientNeverInvoked(ctrl)
		filterCli := filterOutputClientNeverInvoked(ctrl)

		d := Downloader{
			FilterCli:  filterCli,
			DatasetCli: datasetCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(ctx, testImageDownloadParams, TypeImage, testVariant)

		So(downloads.IsPublished, ShouldBeFalse)
		So(downloads.Public, ShouldBeBlank)
		So(downloads.PrivateFilename, ShouldBeBlank)
		So(downloads.PrivateS3Path, ShouldBeBlank)
		So(err, ShouldResemble, errImage)
	})

	Convey("should return publish false if image not published", t, func() {
		imgDownload := getTestImageDownloadImported()

		imgCli := successfulImageClient(ctrl, testImageDownloadParams, imgDownload)
		datasetCli := datasetClientNeverInvoked(ctrl)
		filterCli := filterOutputClientNeverInvoked(ctrl)

		d := Downloader{
			FilterCli:  filterCli,
			DatasetCli: datasetCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(ctx, testImageDownloadParams, TypeImage, testVariant)

		So(downloads.IsPublished, ShouldBeFalse)
		So(downloads.Public, ShouldBeBlank)
		So(downloads.PrivateFilename, ShouldResemble, testImagePrivateFilename)
		So(downloads.PrivateS3Path, ShouldResemble, testImagePrivateS3Path)
		So(err, ShouldBeNil)
	})

	Convey("should return expected values if image is published", t, func() {
		imgDownload := getTestImageDownloadPublished()

		imgCli := successfulImageClient(ctrl, testImageDownloadParams, imgDownload)
		datasetCli := datasetClientNeverInvoked(ctrl)
		filterCli := filterOutputClientNeverInvoked(ctrl)

		d := Downloader{
			FilterCli:  filterCli,
			DatasetCli: datasetCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(ctx, testImageDownloadParams, TypeImage, testVariant)

		So(downloads.IsPublished, ShouldBeTrue)
		So(downloads.Public, ShouldBeBlank)
		So(downloads.PrivateFilename, ShouldResemble, testImagePrivateFilename)
		So(downloads.PrivateS3Path, ShouldResemble, testImagePrivateS3Path)
		So(err, ShouldBeNil)
	})

	Convey("should return expected values if image is completed", t, func() {
		imgDownload := getTestImageDownloadCompleted()

		imgCli := successfulImageClient(ctrl, testImageDownloadParams, imgDownload)
		datasetCli := datasetClientNeverInvoked(ctrl)
		filterCli := filterOutputClientNeverInvoked(ctrl)

		d := Downloader{
			FilterCli:  filterCli,
			DatasetCli: datasetCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(ctx, testImageDownloadParams, TypeImage, testVariant)

		So(downloads.IsPublished, ShouldBeTrue)
		So(downloads.Public, ShouldResemble, testImagePublicUrl)
		So(downloads.PrivateFilename, ShouldResemble, testImagePrivateFilename)
		So(downloads.PrivateS3Path, ShouldResemble, testImagePrivateS3Path)
		So(err, ShouldBeNil)
	})
}

func getTestImageDownloadImported() image.ImageDownload {
	return image.ImageDownload{
		State:  "imported",
		Href:   testImagePrivateUrl,
		Size:   666,
		Public: false,
	}
}

func getTestImageDownloadPublished() image.ImageDownload {
	return image.ImageDownload{
		State:  "published",
		Href:   testImagePrivateUrl,
		Size:   666,
		Public: false,
	}
}

func getTestImageDownloadCompleted() image.ImageDownload {
	return image.ImageDownload{
		State:  "completed",
		Href:   testImagePublicUrl,
		Size:   666,
		Public: true,
	}
}

func erroringImageClient(c *gomock.Controller, p Parameters, err error) *mocks.MockImageClient {
	imgCli := mocks.NewMockImageClient(c)

	imgCli.EXPECT().GetDownloadVariant(
		gomock.Any(),
		gomock.Eq(p.UserAuthToken),
		gomock.Eq(p.ServiceAuthToken),
		gomock.Eq(p.CollectionID),
		gomock.Eq(p.ImageID),
		gomock.Eq(p.Variant),
	).Times(1).Return(image.ImageDownload{}, err)

	return imgCli
}

func successfulImageClient(c *gomock.Controller, p Parameters, img image.ImageDownload) *mocks.MockImageClient {
	imgCli := mocks.NewMockImageClient(c)

	imgCli.EXPECT().GetDownloadVariant(
		gomock.Any(),
		gomock.Eq(p.UserAuthToken),
		gomock.Eq(p.ServiceAuthToken),
		gomock.Eq(p.CollectionID),
		gomock.Eq(p.ImageID),
		gomock.Eq(p.Variant),
	).Times(1).Return(img, nil)

	return imgCli
}

func imageClientNeverInvoked(c *gomock.Controller) *mocks.MockImageClient {
	imgCli := mocks.NewMockImageClient(c)

	imgCli.EXPECT().GetDownloadVariant(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Times(0).Return(image.ImageDownload{}, nil)

	return imgCli
}
