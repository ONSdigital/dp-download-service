package api

import (
	"fmt"
	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/files"
	"github.com/ONSdigital/dp-net/v2/request"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/gorilla/mux"
	"io"
	"net/http"
)

// DoDownload handles download generic file requests.
func CreateV1DownloadHandler(fetchMetadata files.MetadataFetcher, downloadFile files.FileDownloader, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {

		callerPresent := request.IsCallerPresent(req.Context())
		fmt.Printf("caller present: %v\n", callerPresent)
		fmt.Printf("caller identity: %v\n", req.Context().Value(request.CallerIdentityKey))
		fmt.Printf("is publishing mode: %v\n", cfg.IsPublishing)

		filePath := mux.Vars(req)["path"]

		metadata, err := fetchMetadata(filePath)
		if err != nil {
			handleError(w, err)
			return
		}

		// if its not UPLOADED && IS_PUBLISHING && IS CALLER PRESENT
			if metadata.Decrypted() {
				log.Info(req.Context(), "File already decrypted, redirecting")
				w.Header().Set("Location", cfg.PublicBucketURL.String() + filePath)
				w.WriteHeader(http.StatusMovedPermanently)
				return
			}

		if isWebMode(cfg) {
			if metadata.Unpublished() {
				log.Info(req.Context(), "File is not published yet")
				w.WriteHeader(http.StatusNotFound)
				return
			}
		}

		setHeaders(w, metadata)

		file, err := downloadFile(filePath)
		if err != nil {
			handleError(w, err)
			return
		}

		defer func() {
			if err := file.Close(); err != nil {
				log.Error(req.Context(), "error closing io.Closer", err)
			}
		}()

		_, err = io.Copy(w, file)
		if err != nil {
			log.Error(req.Context(), "failed to stream file content", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

func isWebMode(cfg *config.Config) bool {
	return cfg.IsPublishing == false
}

func setHeaders(w http.ResponseWriter, m files.Metadata) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", m.GetContentLength())
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", m.GetFilename()))
}
