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
	testUserAuthToken    = "userAuthToken"
	testServiceAuthToken = "serviceAuthToken"
	testCollectionID     = "collectionID"
	testImageID          = "myImageID"
	testVariant          = "1280x720"
	testName             = "myImageName"
	testExt              = "png"
)

var (
	testErrImage = errors.New("borked image")

	testImageDownloadParams = Parameters{
		ImageID: testImageID,
		Variant: testVariant,
		Name:    testName,
		Ext:     testExt,
	}
)

func TestGetDownloadsForImage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	Convey("should return the error if image client get image is unsuccessful", t, func() {
		imgCli := erroringImageClient(ctrl, testImageDownloadParams, testErrImage)
		datasetCli := datasetClientNeverInvoked(ctrl)
		filterCli := filterOutputClientNeverInvoked(ctrl)

		d := Downloader{
			FilterCli:  filterCli,
			DatasetCli: datasetCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(nil, testImageDownloadParams, TypeImage)

		So(downloads.Available, ShouldHaveLength, 0)
		So(downloads.IsPublished, ShouldBeFalse)
		So(err, ShouldResemble, testErrImage)
	})

	Convey("should return publish false if image not published", t, func() {
		imgDownload := getTestImageDownload(false)
		image := getTestImage(false, &imgDownload)

		imgCli := successfulImageClient(ctrl, testImageDownloadParams, image)
		datasetCli := datasetClientNeverInvoked(ctrl)
		filterCli := filterOutputClientNeverInvoked(ctrl)

		d := Downloader{
			FilterCli:  filterCli,
			DatasetCli: datasetCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(nil, testImageDownloadParams, TypeImage)

		So(downloads.Available, ShouldHaveLength, 1)
		img, found := downloads.Available[testVariant]
		So(found, ShouldBeTrue)

		So(img, ShouldResemble, Info{
			Public:  imgDownload.Href,
			Private: imgDownload.Private,
		})

		So(downloads.IsPublished, ShouldBeFalse)
		So(err, ShouldBeNil)
	})

	Convey("should return expected values if downloads is not empty", t, func() {
		imgDownload := getTestImageDownload(true)
		img := getTestImage(true, &imgDownload)

		imgCli := successfulImageClient(ctrl, testImageDownloadParams, img)
		datasetCli := datasetClientNeverInvoked(ctrl)
		filterCli := filterOutputClientNeverInvoked(ctrl)

		d := Downloader{
			FilterCli:  filterCli,
			DatasetCli: datasetCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(nil, testImageDownloadParams, TypeImage)

		So(downloads.Available, ShouldHaveLength, 1)

		csv, found := downloads.Available[testVariant]
		So(found, ShouldBeTrue)

		So(csv, ShouldResemble, Info{
			Public:  imgDownload.Href,
			Private: imgDownload.Private,
		})

		So(downloads.IsPublished, ShouldBeTrue)
		So(err, ShouldBeNil)
	})

	Convey("should return expected values if downloads is empty", t, func() {
		img := getTestImage(true, nil)

		imgCli := successfulImageClient(ctrl, testImageDownloadParams, img)
		datasetCli := datasetClientNeverInvoked(ctrl)
		filterCli := filterOutputClientNeverInvoked(ctrl)

		d := Downloader{
			FilterCli:  filterCli,
			DatasetCli: datasetCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(nil, testImageDownloadParams, TypeImage)

		So(downloads.Available, ShouldHaveLength, 0)
		So(downloads.IsPublished, ShouldBeTrue)
		So(err, ShouldBeNil)
	})
}

func getTestImageDownload(isPublic bool) image.ImageDownload {
	return image.ImageDownload{
		Href:    "/downloadURL",
		Size:    666,
		Public:  isPublic,
		Private: "/private/download/url",
	}
}

func getTestImage(isPublished bool, dl *image.ImageDownload) image.Image {
	i := image.Image{State: "created"}
	if isPublished {
		i.State = "published"
	}

	if dl != nil {
		i.Downloads = map[string]image.ImageDownload{
			testVariant: *dl,
		}
	}
	return i
}

func erroringImageClient(c *gomock.Controller, p Parameters, err error) *mocks.MockImageClient {
	imgCli := mocks.NewMockImageClient(c)

	imgCli.EXPECT().GetImage(
		nil,
		gomock.Eq(p.UserAuthToken),
		gomock.Eq(p.ServiceAuthToken),
		gomock.Eq(p.CollectionID),
		gomock.Eq(p.ImageID),
	).Times(1).Return(image.Image{}, err)

	return imgCli
}

func successfulImageClient(c *gomock.Controller, p Parameters, img image.Image) *mocks.MockImageClient {
	imgCli := mocks.NewMockImageClient(c)

	imgCli.EXPECT().GetImage(
		nil,
		gomock.Eq(p.UserAuthToken),
		gomock.Eq(p.ServiceAuthToken),
		gomock.Eq(p.CollectionID),
		gomock.Eq(p.ImageID),
	).Times(1).Return(img, nil)

	return imgCli
}

func imageClientNeverInvoked(c *gomock.Controller) *mocks.MockImageClient {
	imgCli := mocks.NewMockImageClient(c)

	imgCli.EXPECT().GetImage(
		nil,
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Times(0).Return(image.Image{}, nil)

	return imgCli
}
