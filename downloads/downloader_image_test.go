package downloads

import (
	"errors"
	"testing"

	"github.com/ONSdigital/dp-api-clients-go/image"
	"github.com/ONSdigital/dp-download-service/downloads/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	testImageID = "myImageID"
	testVariant = "myVariant"
	testName    = "myImageName.png"

	testImagePublicUrl        = "http://public.localhost/images/myImageID/myVariant/myImageName.png"
	testImagePrivateUrl       = "http://private.localhost/images/myImageID/myVariant/myImageName.png"
	testImagePrivateFilename  = "myImageName.png"
	testImagePrivateS3Path    = "/images/myImageID/myVariant"
	testImagePrivateVaultPath = "/images/myImageID/myVariant"
)

var (
	testErrImage = errors.New("borked image")

	testImageDownloadParams = Parameters{
		ImageID:  testImageID,
		Variant:  testVariant,
		Filename: testName,
	}
)

func TestGetDownloadsForImage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	Convey("should return the error if image client get image is unsuccessful (eg. 404 Not Found)", t, func() {
		imgCli := erroringImageClient(ctrl, testImageDownloadParams, testErrImage)
		datasetCli := datasetClientNeverInvoked(ctrl)
		filterCli := filterOutputClientNeverInvoked(ctrl)

		d := Downloader{
			FilterCli:  filterCli,
			DatasetCli: datasetCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(nil, testImageDownloadParams, TypeImage, testVariant)

		So(downloads.IsPublished, ShouldBeFalse)
		So(downloads.Public, ShouldBeBlank)
		So(downloads.PrivateFilename, ShouldBeBlank)
		So(downloads.PrivateS3Path, ShouldBeBlank)
		So(downloads.PrivateVaultPath, ShouldBeBlank)
		So(err, ShouldResemble, testErrImage)
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

		downloads, err := d.Get(nil, testImageDownloadParams, TypeImage, testVariant)

		So(downloads.IsPublished, ShouldBeFalse)
		So(downloads.Public, ShouldBeBlank)
		So(downloads.PrivateFilename, ShouldResemble, testImagePrivateFilename)
		So(downloads.PrivateS3Path, ShouldResemble, testImagePrivateS3Path)
		So(downloads.PrivateVaultPath, ShouldResemble, testImagePrivateVaultPath)
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

		downloads, err := d.Get(nil, testImageDownloadParams, TypeImage, testVariant)

		So(downloads.IsPublished, ShouldBeTrue)
		So(downloads.Public, ShouldBeBlank)
		So(downloads.PrivateFilename, ShouldResemble, testImagePrivateFilename)
		So(downloads.PrivateS3Path, ShouldResemble, testImagePrivateS3Path)
		So(downloads.PrivateVaultPath, ShouldResemble, testImagePrivateVaultPath)
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

		downloads, err := d.Get(nil, testImageDownloadParams, TypeImage, testVariant)

		So(downloads.IsPublished, ShouldBeTrue)
		So(downloads.Public, ShouldResemble, testImagePublicUrl)
		So(downloads.PrivateFilename, ShouldResemble, testImagePrivateFilename)
		So(downloads.PrivateS3Path, ShouldResemble, testImagePrivateS3Path)
		So(downloads.PrivateVaultPath, ShouldResemble, testImagePrivateVaultPath)
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
		nil,
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
		nil,
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
		nil,
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Times(0).Return(image.ImageDownload{}, nil)

	return imgCli
}
