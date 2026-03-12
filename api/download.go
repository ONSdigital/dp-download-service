package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	auth "github.com/ONSdigital/dp-authorisation/v2/authorisation"
	dprequest "github.com/ONSdigital/dp-net/v3/request"

	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/files"
	filesAPIModels "github.com/ONSdigital/dp-files-api/files"
	filesAPISDK "github.com/ONSdigital/dp-files-api/sdk"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/gorilla/mux"
)

// CreateV1DownloadHandler handles generic download file requests.
func CreateV1DownloadHandler(fetchMetadata files.MetadataFetcher, downloadFileFromBucket files.FileDownloader, createFileEvent files.FileEventCreator, authMiddleware auth.Middleware, cfg *config.Config, permissionsChecker auth.PermissionsChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if cfg.IsPublishing {
			w.Header().Set("Cache-Control", "no-cache")
		}

		accessToken := getAccessTokenFromRequest(req)

		ctx, requestedFilePath := parseRequest(req)
		log.Info(ctx, fmt.Sprintf("Handling request for %s", requestedFilePath))

		metadata, err := fetchMetadata(ctx, requestedFilePath, filesAPISDK.Headers{Authorization: accessToken})
		if err != nil {
			switch {
			case strings.Contains(err.Error(), files.ErrFileNotRegistered.Error()):
				handleError(ctx, "Error fetching metadata", w, files.ErrFileNotRegistered)
			case strings.Contains(err.Error(), files.ErrInvalidAuth.Error()):
				handleError(ctx, "Error fetching metadata", w, files.ErrInvalidAuth)
			case strings.Contains(err.Error(), files.ErrNotAuthorised.Error()):
				handleError(ctx, "Error fetching metadata", w, files.ErrNotAuthorised)
			default:
				handleError(ctx, "Error fetching metadata", w, err)
			}
			return
		}

		log.Info(ctx, fmt.Sprintf("Found metadata for file %s", requestedFilePath), log.Data{"metadata": metadata})

		if cfg.IsPublishing {
			logData := log.Data{
				"method": req.Method,
				"path":   req.URL.Path,
			}
			entityData, err := getAuthEntityData(ctx, authMiddleware, accessToken, logData)
			if err != nil {
				log.Error(req.Context(), "the request was not authorised", err, logData)
				if strings.Contains(err.Error(), "key id unknown or invalid") || strings.Contains(err.Error(), "jwt token is malformed") || strings.Contains(err.Error(), "unable to parse jwt") {
					handleError(ctx, "Unauthorised", w, files.ErrInvalidAuth)
					return
				}
				handleError(ctx, "the request was not authorised - check token and user's permissions", w, err)
				return
			}

			var permissionAttrs map[string]string
			if metadata.ContentItem != nil {
				if metadata.ContentItem.DatasetID != "" && metadata.ContentItem.Edition != "" {
					permissionAttrs = map[string]string{
						"dataset_edition": metadata.ContentItem.DatasetID + "/" + metadata.ContentItem.Edition,
					}
				}
			}

			logData = log.Data{
				"entity_data": entityData,
			}

			if checkUserPermission(req.Context(), logData, "static-files:read", permissionAttrs, permissionsChecker, entityData) {
				// Passing identifier as both user and email parameters as the identity client only provides a single identifier
				auditEvent, err := files.PopulateFileEvent(entityData.UserID, entityData.UserID, requestedFilePath, filesAPIModels.ActionRead, metadata)
				if err != nil {
					handleError(ctx, "Failed to populate file event", w, err)
					return
				}

				_, err = createFileEvent(ctx, auditEvent, filesAPISDK.Headers{Authorization: accessToken})
				if err != nil {
					handleError(ctx, "Failed to create file event", w, err)
					return
				}
			} else {
				log.Info(req.Context(), "authorisation failed: request has no permission", log.Classification(log.ProtectiveMonitoring), log.Auth(log.USER, entityData.UserID), logData)
				handleError(ctx, "the request was not authorised - check token and user's permissions", w, files.ErrNotAuthorised)
				return
			}
		}

		if handleUnsupportedMetadataStates(ctx, *metadata, cfg, requestedFilePath, w) {
			return
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
