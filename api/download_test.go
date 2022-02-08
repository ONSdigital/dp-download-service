package api_test

import (
	"errors"
	"github.com/ONSdigital/dp-download-service/api"
	"github.com/ONSdigital/dp-download-service/files"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"testing"
)

type ErrorWriter struct {
	status int
}

func (e *ErrorWriter) Header() http.Header {
	return http.Header{}
}

func (e *ErrorWriter) Write(i []byte) (int, error) {
	return 0, errors.New("broken")
}

func (e *ErrorWriter) WriteHeader(statusCode int) {
	e.status = statusCode
}

type DummyReadCloser struct{}

func (d DummyReadCloser) Read(p []byte) (n int, err error) {
	return 0, errors.New("broken")
}

func (d DummyReadCloser) Close() error {
	return nil
}

func TestHandlingErrorSteamingFileContent(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/v1/files/data/file.csv", nil)
	rec := &ErrorWriter{}

	h := api.CreateV1DownloadHandler(func(path string) (files.Metadata, io.ReadCloser, error) {
		return files.Metadata{State: "PUBLISHED"}, DummyReadCloser{}, nil
	})

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.status)
}

func TestHandleFileNotPublished(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/v1/files/data/file.csv", nil)
	rec := &ErrorWriter{}

	type args struct {
		retrieve files.FileDownloader
	}
	tests := []struct {
		name           string
		expectedStatus int
		state          files.State
	}{
		{"Test CREATED", http.StatusNotFound, files.CREATED},
		{"Test UPDATED", http.StatusNotFound, files.UPLOADED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			h := api.CreateV1DownloadHandler(func(path string) (files.Metadata, io.ReadCloser, error) {
				return files.Metadata{State: tt.state}, DummyReadCloser{}, nil
			})

			h.ServeHTTP(rec, req)

			assert.Equalf(t, tt.expectedStatus, rec.status, "CreateV1DownloadHandler(%v)", tt.name)
		})
	}
}
