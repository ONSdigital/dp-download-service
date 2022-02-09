package api

import (
	"fmt"
	"github.com/ONSdigital/dp-download-service/files"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	"net/url"
	"path"
)

// DoDownload handles download generic file requests.
func CreateV1DownloadHandler(fetchMetadata files.MetadataFetcher, downloadFile files.FileDownloader, publicBucketURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		filePath := mux.Vars(req)["path"]

		metadata, err := fetchMetadata(filePath)
		if err != nil {
			handleError(w, err)
			return
		}

		if metadata.Decrypted() {
			log.Info(req.Context(), "File already decrypted, redirecting")
			parsedURL, err := url.Parse(publicBucketURL)

			if err != nil {
				log.Error(req.Context(), fmt.Sprintf("Bad public bucket url: %s", publicBucketURL), err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			parsedURL.Path = path.Join(filePath)
			w.Header().Set("Location", parsedURL.String())
			w.WriteHeader(http.StatusMovedPermanently)
			return
		}

		if metadata.Unpublished() {
			log.Info(req.Context(), "File is not published yet")
			w.WriteHeader(http.StatusNotFound)
			return
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

func setHeaders(w http.ResponseWriter, m files.Metadata) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", m.GetContentLength())
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", m.GetFilename()))
}
