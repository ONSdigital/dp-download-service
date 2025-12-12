package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	iclient "github.com/ONSdigital/dp-api-clients-go/v2/identity"
	dprequest "github.com/ONSdigital/dp-net/v3/request"

	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/downloads"
	"github.com/ONSdigital/dp-download-service/files"
	filesAPIModels "github.com/ONSdigital/dp-files-api/files"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/gorilla/mux"
)

// CreateV1DownloadHandler handles generic download file requests.
func CreateV1DownloadHandler(fetchMetadata files.MetadataFetcher, downloadFileFromBucket files.FileDownloader, filesClient downloads.FilesClient, identityClient *iclient.Client, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if cfg.IsPublishing {
			w.Header().Set("Cache-Control", "no-cache")
		}

		ctx, requestedFilePath := parseRequest(req)
		log.Info(ctx, fmt.Sprintf("Handling request for %s", requestedFilePath))

		metadata, err := fetchMetadata(ctx, requestedFilePath)
		if err != nil {
			if strings.Contains(err.Error(), files.ErrFileNotRegistered.Error()) {
				handleError(ctx, "Error fetching metadata", w, files.ErrFileNotRegistered)
				return
			}
			handleError(ctx, "Error fetching metadata", w, err)
			return
		}
		log.Info(ctx, fmt.Sprintf("Found metadata for file %s", requestedFilePath), log.Data{"metadata": metadata})

		if handleUnsupportedMetadataStates(ctx, *metadata, cfg, requestedFilePath, w) {
			return
		}

		if cfg.IsPublishing {
			authToken := req.Header.Get(dprequest.AuthHeaderKey)
			if authToken != "" {
				if err := LogFileEvent(ctx, filesClient, identityClient, req, requestedFilePath, metadata); err != nil {
					handleError(ctx, "Failed to log file event", w, err)
					return
				}
			}
		}

		setContentHeaders(w, *metadata)

		file, err := downloadFileFromBucket(requestedFilePath)
		if err != nil {
			handleError(ctx, fmt.Sprintf("Error downloading file %s", requestedFilePath), w, err)
			return
		}

		defer closeDownloadedFile(ctx, file)

		err = writeFileToResponse(w, file)
		if err != nil {
			log.Error(ctx, "Failed to stream file content", err)
			setStatusInternalServerError(w)
			return
		}
	}
}

func LogFileEvent(ctx context.Context, filesClient downloads.FilesClient, identityClient *iclient.Client, req *http.Request, filePath string, metadata *filesAPIModels.StoredRegisteredMetaData) error {
	requestedBy := GetUserIdentity(ctx, identityClient, req)

	event := filesAPIModels.FileEvent{
		RequestedBy: requestedBy,
		Action:      filesAPIModels.ActionRead,
		Resource:    req.URL.Path,
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
	}

	_, err := filesClient.CreateFileEvent(ctx, event)
	if err != nil {
		log.Error(ctx, "failed to create file event", err, log.Data{
			"file_path": filePath,
			"user_id":   requestedBy.ID,
		})
		return fmt.Errorf("failed to create file event: %w", err)
	}

	return nil
}

func GetUserIdentity(ctx context.Context, identityClient *iclient.Client, req *http.Request) *filesAPIModels.RequestedBy {
	authToken := req.Header.Get(dprequest.AuthHeaderKey)
	authToken = strings.TrimPrefix(authToken, dprequest.BearerPrefix)

	identityResp, err := identityClient.CheckTokenIdentity(ctx, authToken, iclient.TokenTypeUser)
	if err == nil && identityResp != nil {
		return &filesAPIModels.RequestedBy{
			ID:    identityResp.Identifier,
			Email: identityResp.Identifier,
		}
	}

	identityResp, err = identityClient.CheckTokenIdentity(ctx, authToken, iclient.TokenTypeService)
	if err == nil && identityResp != nil {
		return &filesAPIModels.RequestedBy{
			ID:    identityResp.Identifier,
			Email: "",
		}
	}

	log.Error(ctx, "failed to validate token with identity service", err)
	return &filesAPIModels.RequestedBy{
		ID: "unauthorised",
	}
}

func parseRequest(req *http.Request) (context.Context, string) {
	ctx := req.Context()
	filePath := mux.Vars(req)["path"]

	authHeaderValue := req.Header.Get(dprequest.AuthHeaderKey)
	if authHeaderValue != "" {
		const key files.ContextKey = dprequest.AuthHeaderKey
		ctx = context.WithValue(ctx, key, authHeaderValue)
	}

	return ctx, filePath
}

func handleUnsupportedMetadataStates(ctx context.Context, m filesAPIModels.StoredRegisteredMetaData, cfg *config.Config, filePath string, w http.ResponseWriter) bool {
	if files.Moved(&m) {
		log.Info(ctx, "File moved, redirecting")
		setStatusMovedPermanently(RedirectLocation(cfg, filePath), w)
		return true
	}

	if files.UploadIncomplete(&m) {
		log.Info(ctx, "File has not finished uploading")
		setStatusNotFound(w)
		return true
	}

	if files.Unpublished(&m) && isWebMode(cfg) {
		log.Info(ctx, "File is not published yet")
		setStatusNotFound(w)
		return true
	}

	return false
}

func closeDownloadedFile(ctx context.Context, file io.ReadCloser) {
	if err := file.Close(); err != nil {
		log.Error(ctx, "error closing io.Closer for file streaming", err)
	}
}

func setStatusMovedPermanently(location string, w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "max-age=31536000")
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusMovedPermanently)
}

func setStatusNotFound(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusNotFound)
}

func setStatusInternalServerError(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusInternalServerError)
}

func RedirectLocation(cfg *config.Config, filePath string) string {
	redirectURL, err := url.Parse(cfg.PublicBucketURL.String())
	if err != nil {
		ctx := context.Background()
		log.Error(ctx, "error parsing public bucket URL", err)
	}
	redirectURL.Path = path.Join(redirectURL.Path, filePath)
	return redirectURL.String()
}

func writeFileToResponse(w http.ResponseWriter, file io.ReadCloser) error {
	_, err := io.Copy(w, file)
	return err
}

func isWebMode(cfg *config.Config) bool {
	return !cfg.IsPublishing
}

func setContentHeaders(w http.ResponseWriter, m filesAPIModels.StoredRegisteredMetaData) {
	w.Header().Set("Content-Type", m.Type)
	w.Header().Set("Content-Length", files.GetContentLength(&m))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", files.GetFilename(&m)))
}
