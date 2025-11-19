package downloads

import (
	"context"

	"github.com/ONSdigital/dp-api-clients-go/v2/files"
	filesModel "github.com/ONSdigital/dp-files-api/files"
	filesSDK "github.com/ONSdigital/dp-files-api/sdk"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
)

// FilesClientWrapper wraps both the old files client and the new SDK client
type FilesClientWrapper struct {
	filesClient *files.Client
	sdkClient   *filesSDK.Client
}

// NewFilesClientWrapper creates a new wrapper that implements the FilesClient interface
func NewFilesClientWrapper(filesAPIURL, authToken string) *FilesClientWrapper {
	return &FilesClientWrapper{
		filesClient: files.NewAPIClient(filesAPIURL, authToken),
		sdkClient:   filesSDK.New(filesAPIURL, authToken),
	}
}

// GetFile uses the existing files client
func (w *FilesClientWrapper) GetFile(ctx context.Context, path string, authToken string) (files.FileMetaData, error) {
	return w.filesClient.GetFile(ctx, path, authToken)
}

// CreateFileEvent uses the new SDK client
func (w *FilesClientWrapper) CreateFileEvent(ctx context.Context, event filesModel.FileEvent) (*filesModel.FileEvent, error) {
	return w.sdkClient.CreateFileEvent(ctx, event)
}

// Checker uses the existing files client checker
func (w *FilesClientWrapper) Checker(ctx context.Context, state *healthcheck.CheckState) error {
	return w.filesClient.Checker(ctx, state)
}
