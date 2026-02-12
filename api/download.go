package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/ONSdigital/dp-api-clients-go/v2/identity"
	"github.com/ONSdigital/dp-authorisation/v2/authorisation"
	dprequest "github.com/ONSdigital/dp-net/v3/request"
	permissionsAPISDK "github.com/ONSdigital/dp-permissions-api/sdk"

	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/downloads"
	"github.com/ONSdigital/dp-download-service/files"
	filesAPIModels "github.com/ONSdigital/dp-files-api/files"
	filesAPISDK "github.com/ONSdigital/dp-files-api/sdk"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/gorilla/mux"
)

const staticFilesReadPermission = "static-files:read"

// CreateV1DownloadHandler handles generic download file requests.
func CreateV1DownloadHandler(fetchMetadata files.MetadataFetcher, downloadFileFromBucket files.FileDownloader, createFileEvent files.FileEventCreator, identityClient downloads.IdentityClient, authMiddleware authorisation.Middleware, permissionsChecker authorisation.PermissionsChecker, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if cfg.IsPublishing {
			w.Header().Set("Cache-Control", "no-cache")
		}

		accessToken := getAccessTokenFromRequest(req)
		userToken := ""
		if cfg.IsPublishing {
			userToken = getUserTokenFromRequest(req)
		}

		ctx, requestedFilePath := parseRequest(req)
		log.Info(ctx, fmt.Sprintf("Handling request for %s", requestedFilePath))

		filesAPIToken := accessToken
		if cfg.IsPublishing && userToken != "" {
			filesAPIToken = userToken
		}

		metadata, err := fetchMetadata(ctx, requestedFilePath, filesAPISDK.Headers{Authorization: filesAPIToken})
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

		if cfg.IsPublishing && files.Unpublished(metadata) {
			if cfg.AuthorisationConfig != nil && cfg.AuthorisationConfig.Enabled {
				if authMiddleware == nil || permissionsChecker == nil {
					log.Error(ctx, "Authorisation is enabled but middleware or permissions checker is not configured", fmt.Errorf("authorisation dependencies missing"))
					setStatusInternalServerError(w)
					return
				}
				status, err := authoriseUnpublishedFile(ctx, userToken, accessToken, metadata, authMiddleware, permissionsChecker, identityClient)
				if err != nil {
					log.Error(ctx, "Failed to authorise unpublished file", err)
				}
				switch status {
				case http.StatusUnauthorized:
					setStatusUnauthorized(w)
					return
				case http.StatusForbidden:
					setStatusForbidden(w)
					return
				case http.StatusInternalServerError:
					setStatusInternalServerError(w)
					return
				}
			} else if userToken == "" && accessToken == "" {
				setStatusUnauthorized(w)
				return
			}
		}

		setContentHeaders(w, *metadata)

		if cfg.IsPublishing {
			if userToken == "" {
				if files.Unpublished(metadata) && accessToken == "" && (cfg.AuthorisationConfig == nil || !cfg.AuthorisationConfig.Enabled) {
					setStatusUnauthorized(w)
					return
				}
			} else {
				identifier, err := getTokenIdentifier(ctx, userToken, identityClient)
				if err != nil {
					log.Error(ctx, "Failed to get token identifier from access token", err)
					setStatusUnauthorized(w)
					return
				}

				// Passing identifier as both user and email parameters as the identity client only provides a single identifier
				auditEvent, err := files.PopulateFileEvent(identifier, identifier, requestedFilePath, filesAPIModels.ActionRead, metadata)
				if err != nil {
					handleError(ctx, "Failed to populate file event", w, err)
					return
				}

				_, err = createFileEvent(ctx, auditEvent, filesAPISDK.Headers{Authorization: userToken})
				if err != nil {
					handleError(ctx, "Failed to create file event", w, err)
					return
				}
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

func setStatusUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusUnauthorized)
}

func setStatusForbidden(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusForbidden)
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

func authoriseUnpublishedFile(ctx context.Context, userToken, accessToken string, metadata *filesAPIModels.StoredRegisteredMetaData, authMiddleware authorisation.Middleware, permissionsChecker authorisation.PermissionsChecker, identityClient downloads.IdentityClient) (int, error) {
	var (
		entityData *permissionsAPISDK.EntityData
		err        error
	)

	if userToken != "" {
		entityData, err = authMiddleware.Parse(userToken)
		if err != nil || entityData == nil {
			return http.StatusUnauthorized, err
		}
	} else {
		if accessToken == "" {
			return http.StatusUnauthorized, nil
		}
		if identityClient == nil {
			return http.StatusInternalServerError, fmt.Errorf("identity client is not configured")
		}
		identityResp, identityErr := identityClient.CheckTokenIdentity(ctx, accessToken, identity.TokenTypeService)
		if identityErr != nil {
			return http.StatusUnauthorized, identityErr
		}
		entityData = &permissionsAPISDK.EntityData{UserID: identityResp.Identifier}
	}

	attributes, err := datasetEditionAttributes(metadata)
	if err != nil {
		return http.StatusForbidden, err
	}

	hasPermission, err := permissionsChecker.HasPermission(ctx, *entityData, staticFilesReadPermission, attributes)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	if !hasPermission {
		return http.StatusForbidden, nil
	}
	return http.StatusOK, nil
}

func datasetEditionAttributes(metadata *filesAPIModels.StoredRegisteredMetaData) (map[string]string, error) {
	if metadata == nil {
		return nil, fmt.Errorf("missing content item metadata for permissions check")
	}
	datasetID := strings.TrimSpace(metadata.ContentItem.DatasetID)
	if datasetID == "" {
		return nil, fmt.Errorf("missing dataset ID in content item metadata")
	}
	edition := strings.TrimSpace(metadata.ContentItem.Edition)
	datasetEdition := datasetID
	if edition != "" {
		datasetEdition = fmt.Sprintf("%s/%s", datasetID, edition)
	}
	return map[string]string{"dataset_edition": datasetEdition}, nil
}
