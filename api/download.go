package api

import (
	"fmt"
	"github.com/ONSdigital/dp-download-service/files"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/gorilla/mux"
	"io"
	"net/http"
)

// DoDownload handles download generic file requests.
func CreateV1DownloadHandler(fetchMetadata files.MetadataFetcher, downloadFile files.FileRetriever) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)

		metadata, err := fetchMetadata(vars["path"])
		if err != nil {
			handleError(w, err)
			return
		}


		if metadata.Unpublished() {
			log.Info(req.Context(), "File is not published yet")
			w.WriteHeader(http.StatusNotFound)
			return
		}

		setHeaders(w, metadata)

		file, err := downloadFile(vars["path"])

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
