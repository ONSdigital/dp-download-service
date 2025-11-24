package files

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/ONSdigital/dp-download-service/content"
	"github.com/ONSdigital/dp-download-service/downloads"
	filesSDK "github.com/ONSdigital/dp-files-api/files"
)

type HTTPClient interface {
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
}

var ErrFileNotRegistered = errors.New("file not registered")
var ErrBadJSONResponse = errors.New("could not decode JSON response from files api")
var ErrNotAuthorised = errors.New("the request was not authorised - check token and user's permissions")
var ErrInternalServerError = errors.New("internal server error")
var ErrUnknown = errors.New("an unknown error occurred")
var ErrRequest = errors.New("an error occurred making a request to files api")

type FileDownloader func(path string) (io.ReadCloser, error)
type MetadataFetcher func(ctx context.Context, path string) (*filesSDK.StoredRegisteredMetaData, error)

type ContextKey string

func FetchMetadata(filesClient downloads.FilesClient) MetadataFetcher {
	return func(ctx context.Context, path string) (*filesSDK.StoredRegisteredMetaData, error) {
		return filesClient.GetFile(ctx, path)
	}
}

func DownloadFile(ctx context.Context, s3client content.S3Client) FileDownloader {
	return func(filePath string) (io.ReadCloser, error) {
		file, _, err := s3client.Get(ctx, filePath)

		return file, err
	}
}
