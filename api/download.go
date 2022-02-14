package api

import (
	"fmt"
	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/files"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/gorilla/mux"
	"io"
	"net/http"
)

// DoDownload handles download generic file requests.
func CreateV1DownloadHandler(fetchMetadata files.MetadataFetcher, downloadFile files.FileDownloader, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		filePath := mux.Vars(req)["path"]
		log.Info(req.Context(), "Handling request for " + filePath)

		metadata, err := fetchMetadata(filePath)
		if err != nil {
			log.Error(req.Context(), "Error fetching metadata" , err)
			handleError(w, err)
			return
		}

		log.Info(req.Context(), "Found metadata for file ", log.Data{"metadata": metadata})

		if metadata.Decrypted() {
			log.Info(req.Context(), "File already decrypted, redirecting")
			w.Header().Set("Location", cfg.PublicBucketURL.String()+filePath)
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
			log.Error(req.Context(), "Error downloading file", err)
			handleError(w, err)
			return
		}

		defer func() {
			if err := file.Close(); err != nil {
				log.Error(req.Context(), "error closing io.Closer for file streaming", err)
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
	return !cfg.IsPublishing
}

func setHeaders(w http.ResponseWriter, m files.Metadata) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", m.GetContentLength())
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", m.GetFilename()))
}
