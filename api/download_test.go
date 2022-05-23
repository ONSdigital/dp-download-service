package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ONSdigital/dp-download-service/api"
	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/files"
	"github.com/stretchr/testify/assert"
)

type ErrorWriter struct {
	status int
	header http.Header
}

func (e *ErrorWriter) Header() http.Header {
	return e.header
}

func (e *ErrorWriter) Write(i []byte) (int, error) {
	return 0, errors.New("broken")
}

func (e *ErrorWriter) WriteHeader(statusCode int) {
	e.status = statusCode
}

type FailingReadCloser struct{}

func (d FailingReadCloser) Read(p []byte) (n int, err error) {
	return 0, errors.New("broken")
}

func (d FailingReadCloser) Close() error {
	return nil
}

func TestHandlingErrorForMetadata(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", nil)
	rec := &ErrorWriter{header: make(http.Header)}

	fetchMetadata := func(ctx context.Context, path string) (files.Metadata, error) {
		return files.Metadata{State: "PUBLISHED"}, nil
	}
	downloadFile := func(path string) (io.ReadCloser, error) { return FailingReadCloser{}, nil }

	h := api.CreateV1DownloadHandler(fetchMetadata, downloadFile, &config.Config{})

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.status)
	assert.Equal(t, rec.Header().Get("Cache-Control"), "no-cache")
}

func TestHandlingAuthErrorFetchingMetadata(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", nil)
	rec := &ErrorWriter{header: make(http.Header)}

	fetchMetadata := func(ctx context.Context, path string) (files.Metadata, error) {
		return files.Metadata{}, files.ErrNotAuthorised
	}
	downloadFile := func(path string) (io.ReadCloser, error) { return nil, nil }

	h := api.CreateV1DownloadHandler(fetchMetadata, downloadFile, &config.Config{})

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.status)
	assert.Equal(t, rec.Header().Get("Cache-Control"), "no-cache")
}

func TestHandlingUnexpectedErrorFetchingMetadata(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", nil)
	rec := &ErrorWriter{header: make(http.Header)}

	fetchMetadata := func(ctx context.Context, path string) (files.Metadata, error) {
		return files.Metadata{}, files.ErrUnknown
	}
	downloadFile := func(path string) (io.ReadCloser, error) { return nil, nil }

	h := api.CreateV1DownloadHandler(fetchMetadata, downloadFile, &config.Config{})

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.status)
}

func TestHandlingErrorGettingFileContent(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", nil)
	rec := &ErrorWriter{header: make(http.Header)}

	fetchMetadata := func(ctx context.Context, path string) (files.Metadata, error) {
		return files.Metadata{State: "PUBLISHED"}, nil
	}
	downloadFile := func(path string) (io.ReadCloser, error) { return nil, errors.New("error downloading file") }

	h := api.CreateV1DownloadHandler(fetchMetadata, downloadFile, &config.Config{})

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.status)
	assert.Equal(t, rec.Header().Get("Cache-Control"), "no-cache")
}

func TestHandleFileNotPublished(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", nil)
	rec := &ErrorWriter{header: make(http.Header)}

	type args struct {
		retrieve files.FileDownloader
	}
	tests := []struct {
		name           string
		expectedStatus int
		state          files.State
	}{
		{"Test CREATED", http.StatusNotFound, files.CREATED},
		{"Test UPLOADED", http.StatusNotFound, files.UPLOADED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetchMetadata := func(ctx context.Context, path string) (files.Metadata, error) {
				return files.Metadata{State: tt.state}, nil
			}
			downloadFile := func(path string) (io.ReadCloser, error) { return nil, nil }

			h := api.CreateV1DownloadHandler(fetchMetadata, downloadFile, &config.Config{})

			h.ServeHTTP(rec, req)

			assert.Equalf(t, tt.expectedStatus, rec.status, "CreateV1DownloadHandler(%v)", tt.name)
			assert.Equal(t, rec.Header().Get("Cache-Control"), "no-cache")
		})
	}
}

func TestHandleFileNotPublishedInPublishingMode(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", nil)

	t.Run("Test CREATED", func(t *testing.T) {
		rec := &ErrorWriter{header: make(http.Header)}
		fetchMetadata := func(ctx context.Context, path string) (files.Metadata, error) {
			return files.Metadata{State: files.CREATED}, nil
		}
		downloadFile := func(path string) (io.ReadCloser, error) { return FailingReadCloser{}, nil }

		h := api.CreateV1DownloadHandler(fetchMetadata, downloadFile, &config.Config{IsPublishing: true})

		h.ServeHTTP(rec, req)

		assert.Equalf(t, http.StatusNotFound, rec.status, "CreateV1DownloadHandler(%v)", "Test CREATED")
		assert.Equal(t, rec.Header().Get("Cache-Control"), "no-cache")
	})

	t.Run("Test UPLOADED", func(t *testing.T) {
		rec := httptest.NewRecorder()
		fetchMetadata := func(ctx context.Context, path string) (files.Metadata, error) {
			return files.Metadata{State: files.UPLOADED}, nil
		}
		downloadFile := func(path string) (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("testing")), nil }

		h := api.CreateV1DownloadHandler(fetchMetadata, downloadFile, &config.Config{IsPublishing: true})

		h.ServeHTTP(rec, req)

		assert.Equalf(t, http.StatusOK, rec.Code, "CreateV1DownloadHandler(%v)", "Test UPLOADED")
	})
}

func TestContentTypeHeader(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", nil)
	rec := httptest.NewRecorder()

	expectedType := "text/csv"

	fetchMetadata := func(ctx context.Context, path string) (files.Metadata, error) {
		return files.Metadata{Type: expectedType, State: files.PUBLISHED}, nil
	}
	downloadFile := func(path string) (io.ReadCloser, error) { return FailingReadCloser{}, nil }

	h := api.CreateV1DownloadHandler(fetchMetadata, downloadFile, &config.Config{})

	h.ServeHTTP(rec, req)

	assert.Equal(t, expectedType, rec.Header().Get("Content-Type"))
}
