package model_test

import (
	"context"
	"io"
	"testing"

	"github.com/ONSdigital/dp-download-service/model"
	"github.com/ONSdigital/dp-download-service/storage"
	"github.com/google/uuid"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCreate(t *testing.T) {
	Convey("Setting up dependencies", t, func() {

		// Set up happy path clients and dependencies.
		//

		mockedStorage := &StorageMock{
			CreateDatasetFunc: func(ctx context.Context, payload *storage.DatasetDocument) error {
				return nil
			},
		}
		ds := model.New(mockedStorage)

		ctx := context.Background()

		Convey("happy path", func() {
			id, err := ds.Create(ctx, &model.DatasetDocument{})

			Convey("should return success", func() {
				So(err, ShouldBeNil)
				_, err := uuid.Parse(id) // just validate uuid format
				So(err, ShouldBeNil)
			})
		})

		Convey("storage error", func() {
			mockedStorage.CreateDatasetFunc = func(ctx context.Context, payload *storage.DatasetDocument) error {
				return io.ErrUnexpectedEOF // an arbitrary error for testing
			}
			uuid, err := ds.Create(ctx, &model.DatasetDocument{})

			Convey("should return error", func() {
				So(err, ShouldEqual, io.ErrUnexpectedEOF)
				So(uuid, ShouldEqual, "")
			})
		})
	})
}
