package downloads

import (
	"context"
	"errors"
	"testing"

	"github.com/ONSdigital/dp-api-clients-go/v2/dataset"
	"github.com/ONSdigital/dp-api-clients-go/v2/headers"
	"github.com/ONSdigital/dp-download-service/downloads/mocks"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	ctx                        = context.Background()
	errInstance                = errors.New("borked instance")
	testInstanceDownloadParams = Parameters{
		UserAuthToken:    "userAuthToken",
		ServiceAuthToken: "serviceAuthToken",
		InstanceID:       "instanceID",
	}
)

func TestGetDownloadForInstance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	Convey("should return error if dataset client get version returns an error", t, func() {
		datasetCli := erroringDatasetClientGetInstance(ctrl, testInstanceDownloadParams, errInstance)
		filterCli := filterOutputClientNeverInvoked(ctrl)
		imgCli := imageClientNeverInvoked(ctrl)

		d := Downloader{
			DatasetCli: datasetCli,
			FilterCli:  filterCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(ctx, testInstanceDownloadParams, TypeInstance, "csv")

		So(downloads.IsPublished, ShouldBeFalse)
		So(downloads.Public, ShouldBeBlank)
		So(downloads.PrivateFilename, ShouldBeBlank)
		So(downloads.PrivateS3Path, ShouldBeBlank)
		So(downloads.PrivateVaultPath, ShouldBeBlank)
		So(err, ShouldResemble, errInstance)
	})

	Convey("should return published=false if instance state is not published", t, func() {
		datasetDownload := testDatasetDownloadBadURL()
		i := testInstance("not published", &datasetDownload)

		datasetCli := successfulDatasetClientGetInstance(ctrl, testInstanceDownloadParams, i)
		filterCli := filterOutputClientNeverInvoked(ctrl)
		imgCli := imageClientNeverInvoked(ctrl)

		d := Downloader{
			DatasetCli: datasetCli,
			FilterCli:  filterCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(ctx, testInstanceDownloadParams, TypeInstance, "csv")

		So(downloads.IsPublished, ShouldBeFalse)
		So(downloads.Public, ShouldResemble, testCSVPublicUrl)
		So(downloads.PrivateFilename, ShouldResemble, "instanceID.csv")
		So(downloads.PrivateS3Path, ShouldResemble, "instances/instanceID.csv")
		So(downloads.PrivateVaultPath, ShouldResemble, "instances/instanceID.csv")
		So(err, ShouldBeNil)
	})

	Convey("should return empty downloads if dataset version downloads empty", t, func() {
		i := testInstance("not published", nil)

		datasetCli := successfulDatasetClientGetInstance(ctrl, testInstanceDownloadParams, i)
		filterCli := filterOutputClientNeverInvoked(ctrl)
		imgCli := imageClientNeverInvoked(ctrl)

		d := Downloader{
			DatasetCli: datasetCli,
			FilterCli:  filterCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(ctx, testInstanceDownloadParams, TypeInstance, "csv")

		So(downloads.IsPublished, ShouldBeFalse)
		So(downloads.Public, ShouldBeBlank)
		So(downloads.PrivateFilename, ShouldBeBlank)
		So(downloads.PrivateS3Path, ShouldBeBlank)
		So(downloads.PrivateVaultPath, ShouldBeBlank)
		So(err, ShouldBeNil)
	})

	Convey("should return published=true if instance state is published", t, func() {
		datasetDownload := testDatasetDownload()
		i := testInstance(dataset.StatePublished.String(), &datasetDownload)

		datasetCli := successfulDatasetClientGetInstance(ctrl, testInstanceDownloadParams, i)
		filterCli := filterOutputClientNeverInvoked(ctrl)
		imgCli := imageClientNeverInvoked(ctrl)

		d := Downloader{
			DatasetCli: datasetCli,
			FilterCli:  filterCli,
			ImageCli:   imgCli,
		}

		downloads, err := d.Get(ctx, testInstanceDownloadParams, TypeInstance, "csv")

		So(downloads.IsPublished, ShouldBeTrue)
		So(downloads.Public, ShouldResemble, testCSVPublicUrl)
		So(downloads.PrivateFilename, ShouldResemble, "instanceID.csv")
		So(downloads.PrivateS3Path, ShouldResemble, "instances/instanceID.csv")
		So(downloads.PrivateVaultPath, ShouldResemble, "instances/instanceID.csv")
		So(err, ShouldBeNil)
	})
}

func erroringDatasetClientGetInstance(c *gomock.Controller, p Parameters, err error) *mocks.MockDatasetClient {
	cli := mocks.NewMockDatasetClient(c)

	cli.EXPECT().GetInstance(
		gomock.Any(),
		gomock.Eq(p.UserAuthToken),
		gomock.Eq(p.ServiceAuthToken),
		gomock.Eq(p.CollectionID),
		gomock.Eq(p.InstanceID),
		gomock.Eq(headers.IfMatchAnyETag),
	).Times(1).Return(dataset.Instance{}, "", err)
	return cli
}

func testInstance(state string, dl *dataset.Download) dataset.Instance {
	i := dataset.Instance{
		Version: dataset.Version{
			State: state,
		},
	}

	if dl != nil {
		i.Downloads = map[string]dataset.Download{"csv": *dl}
	}

	return i
}

func successfulDatasetClientGetInstance(c *gomock.Controller, p Parameters, i dataset.Instance) *mocks.MockDatasetClient {
	cli := mocks.NewMockDatasetClient(c)

	cli.EXPECT().GetInstance(
		gomock.Any(),
		gomock.Eq(p.UserAuthToken),
		gomock.Eq(p.ServiceAuthToken),
		gomock.Eq(p.CollectionID),
		gomock.Eq(p.InstanceID),
		gomock.Eq(headers.IfMatchAnyETag),
	).Times(1).Return(i, "", nil)
	return cli
}
