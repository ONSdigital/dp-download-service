package api

import (
	"fmt"
	"github.com/ONSdigital/dp-download-service/files"
	"github.com/gorilla/mux"
	"io"
	"net/http"
)

// DoDownload handles download generic file requests.
func CreateV1DownloadHandler(retrieve files.FileRetriever) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)

		m, file, _ := retrieve(vars["path"])

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", m.GetContentLength())
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", m.GetFilename()))

		io.Copy(w, file)
	}
}
