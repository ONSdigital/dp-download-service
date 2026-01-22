package files

import (
	"context"
	"errors"

	"github.com/ONSdigital/dp-download-service/downloads"
	filesAPIModels "github.com/ONSdigital/dp-files-api/files"
	filesAPISDK "github.com/ONSdigital/dp-files-api/sdk"
)

var (
	ErrNilMetadata = errors.New("metadata cannot be nil")
)

type FileEventCreator func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error)

// CreateFileEvent returns a function that creates a file event using the provided files API client.
func CreateFileEvent(filesClient downloads.FilesClient) FileEventCreator {
	return func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error) {
		return filesClient.CreateFileEvent(ctx, event, headers)
	}
}

// PopulateFileEvent creates a FileEvent struct populated with the provided parameters.
func PopulateFileEvent(userID, email, filePath, action string, metadata *filesAPIModels.StoredRegisteredMetaData) (filesAPIModels.FileEvent, error) {
	if metadata == nil {
		return filesAPIModels.FileEvent{}, ErrNilMetadata
	}

	return filesAPIModels.FileEvent{
		RequestedBy: &filesAPIModels.RequestedBy{
			ID:    userID,
			Email: email,
		},
		Action:   action,
		Resource: filePath,
		File: &filesAPIModels.FileMetaData{
			Path:          metadata.Path,
			IsPublishable: metadata.IsPublishable,
			CollectionID:  metadata.CollectionID,
			BundleID:      metadata.BundleID,
			Title:         metadata.Title,
			SizeInBytes:   metadata.SizeInBytes,
			Type:          metadata.Type,
			Licence:       metadata.Licence,
			LicenceURL:    metadata.LicenceURL,
			State:         metadata.State,
			Etag:          metadata.Etag,
		},
	}, nil
}
