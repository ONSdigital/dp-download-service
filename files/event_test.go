package files

import (
	"testing"

	filesAPIModels "github.com/ONSdigital/dp-files-api/files"

	. "github.com/smartystreets/goconvey/convey"
)

var (
	testUserID   = "test-user-id"
	testEmail    = "test@test.com"
	testFilePath = "test/path/to/file.csv"
	testAction   = "READ"

	testCollectionID = "test-collection-id"
	testBundleID     = "test-bundle-id"

	exampleStoredRegisteredMetaData = &filesAPIModels.StoredRegisteredMetaData{
		Path:          "some/path/to/data.csv",
		IsPublishable: true,
		CollectionID:  &testCollectionID,
		BundleID:      &testBundleID,
		Title:         "Test Title",
		SizeInBytes:   64,
		Type:          "text/csv",
		Licence:       "Test Licence",
		LicenceURL:    "http://example.com/licence",
		State:         "PUBLISHED",
		Etag:          "test-etag",
	}
)

func TestPopulateFileEvent(t *testing.T) {
	Convey("When PopulateFileEvent is called with all parameters", t, func() {
		event, err := PopulateFileEvent(testUserID, testEmail, testFilePath, testAction, exampleStoredRegisteredMetaData)

		Convey("Then the returned FileEvent is populated correctly", func() {
			So(err, ShouldBeNil)

			So(event.RequestedBy.ID, ShouldEqual, testUserID)
			So(event.RequestedBy.Email, ShouldEqual, testEmail)
			So(event.Action, ShouldEqual, testAction)
			So(event.Resource, ShouldEqual, testFilePath)

			So(event.File.Path, ShouldEqual, exampleStoredRegisteredMetaData.Path)
			So(event.File.IsPublishable, ShouldEqual, exampleStoredRegisteredMetaData.IsPublishable)
			So(event.File.CollectionID, ShouldEqual, exampleStoredRegisteredMetaData.CollectionID)
			So(event.File.BundleID, ShouldEqual, exampleStoredRegisteredMetaData.BundleID)
			So(event.File.Title, ShouldEqual, exampleStoredRegisteredMetaData.Title)
			So(event.File.SizeInBytes, ShouldEqual, exampleStoredRegisteredMetaData.SizeInBytes)
			So(event.File.Type, ShouldEqual, exampleStoredRegisteredMetaData.Type)
			So(event.File.Licence, ShouldEqual, exampleStoredRegisteredMetaData.Licence)
			So(event.File.LicenceURL, ShouldEqual, exampleStoredRegisteredMetaData.LicenceURL)
			So(event.File.State, ShouldEqual, exampleStoredRegisteredMetaData.State)
			So(event.File.Etag, ShouldEqual, exampleStoredRegisteredMetaData.Etag)
		})
	})

	Convey("When PopulateFileEvent is called with nil metadata", t, func() {
		_, err := PopulateFileEvent(testUserID, testEmail, testFilePath, testAction, nil)

		Convey("Then an ErrNilMetadata error is returned", func() {
			So(err, ShouldEqual, ErrNilMetadata)
		})
	})
}
