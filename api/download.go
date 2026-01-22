package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	dprequest "github.com/ONSdigital/dp-net/v3/request"

	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/downloads"
	"github.com/ONSdigital/dp-download-service/files"
	filesAPIModels "github.com/ONSdigital/dp-files-api/files"
	filesAPISDK "github.com/ONSdigital/dp-files-api/sdk"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/gorilla/mux"
)

// CreateV1DownloadHandler handles generic download file requests.
func CreateV1DownloadHandler(fetchMetadata files.MetadataFetcher, downloadFileFromBucket files.FileDownloader, createFileEvent files.FileEventCreator, identityClient downloads.IdentityClient, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if cfg.IsPublishing {
			w.Header().Set("Cache-Control", "no-cache")
		}

		accessToken := getAccessTokenFromHeaders(req.Header)

		ctx, requestedFilePath := parseRequest(req)
		log.Info(ctx, fmt.Sprintf("Handling request for %s", requestedFilePath))

		metadata, err := fetchMetadata(ctx, requestedFilePath, filesAPISDK.Headers{Authorization: accessToken})
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

		setContentHeaders(w, *metadata)

		if cfg.IsPublishing {
			identifier, err := getTokenIdentifier(ctx, accessToken, identityClient)
			if err != nil {
				handleError(ctx, "Failed to get token identifier from access token", w, err)
				return
			}

			// Passing identifier as both user and email parameters as the identity client only provides a single identifier
			auditEvent, err := files.PopulateFileEvent(identifier, identifier, requestedFilePath, filesAPIModels.ActionRead, metadata)
			if err != nil {
				handleError(ctx, "Failed to populate file event", w, err)
				return
			}

			_, err = createFileEvent(ctx, auditEvent, filesAPISDK.Headers{Authorization: accessToken})
			if err != nil {
				handleError(ctx, "Failed to create file event", w, err)
				return
			}
		}

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
