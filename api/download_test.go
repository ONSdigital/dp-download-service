package api_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ONSdigital/dp-api-clients-go/v2/identity"
	"github.com/ONSdigital/dp-download-service/api"
	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/downloads/mocks"
	"github.com/ONSdigital/dp-download-service/files"
	filesAPIModels "github.com/ONSdigital/dp-files-api/files"
	filesAPISDK "github.com/ONSdigital/dp-files-api/sdk"
	dprequest "github.com/ONSdigital/dp-net/v3/request"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var (
	testAccessToken         = "valid-access-token"
	testAuthorizationHeader = dprequest.BearerPrefix + testAccessToken
	testIdentifier          = "test-identifier"
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

func TestHandlingAuthErrorFetchingMetadata(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", nil)
	rec := &ErrorWriter{header: make(http.Header)}

	fetchMetadata := func(ctx context.Context, path string, headers filesAPISDK.Headers) (*filesAPIModels.StoredRegisteredMetaData, error) {
		return nil, files.ErrNotAuthorised
	}

	createFileEvent := func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error) {
		t.Fatal("createFileEvent should not have been called")
		return nil, nil
	}

	downloadFile := func(path string) (io.ReadCloser, error) { return nil, nil }

	h := api.CreateV1DownloadHandler(fetchMetadata, downloadFile, createFileEvent, nil, &config.Config{})
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.status)
	assert.Equal(t, rec.Header().Get("Cache-Control"), "no-cache")
}

func TestHandlingUnexpectedErrorFetchingMetadata(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", nil)
	rec := &ErrorWriter{header: make(http.Header)}

	fetchMetadata := func(ctx context.Context, path string, headers filesAPISDK.Headers) (*filesAPIModels.StoredRegisteredMetaData, error) {
		return nil, files.ErrUnknown
	}

	createFileEvent := func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error) {
		t.Fatal("createFileEvent should not have been called")
		return nil, nil
	}

	downloadFile := func(path string) (io.ReadCloser, error) { return nil, nil }

	h := api.CreateV1DownloadHandler(fetchMetadata, downloadFile, createFileEvent, nil, &config.Config{})
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.status)
}

func TestHandlingErrorGettingFileContent(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", nil)
	rec := &ErrorWriter{header: make(http.Header)}

	fetchMetadata := func(ctx context.Context, path string, headers filesAPISDK.Headers) (*filesAPIModels.StoredRegisteredMetaData, error) {
		return &filesAPIModels.StoredRegisteredMetaData{State: "PUBLISHED"}, nil
	}

	createFileEvent := func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error) {
		t.Fatal("createFileEvent should not have been called")
		return nil, nil
	}

	downloadFile := func(path string) (io.ReadCloser, error) { return nil, errors.New("error downloading file") }

	h := api.CreateV1DownloadHandler(fetchMetadata, downloadFile, createFileEvent, nil, &config.Config{})
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.status)
	assert.Equal(t, rec.Header().Get("Cache-Control"), "no-cache")
}

func TestHandlingErrorGettingFileNotAvailable(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/unavailablefile.csv", nil)
	rec := &ErrorWriter{header: make(http.Header)}

	fetchMetadata := func(ctx context.Context, path string, headers filesAPISDK.Headers) (*filesAPIModels.StoredRegisteredMetaData, error) {
		return nil, files.ErrFileNotRegistered
	}

	createFileEvent := func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error) {
		t.Fatal("createFileEvent should not have been called")
		return nil, nil
	}

	downloadFile := func(path string) (io.ReadCloser, error) { return nil, errors.New("error downloading file") }

	h := api.CreateV1DownloadHandler(fetchMetadata, downloadFile, createFileEvent, nil, &config.Config{})
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.status)
	assert.Equal(t, rec.Header().Get("Cache-Control"), "no-cache")
}

func TestHandleFileNotPublished(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", nil)
	rec := &ErrorWriter{header: make(http.Header)}

	tests := []struct {
		name           string
		expectedStatus int
		state          string
	}{
		{"Test CREATED", http.StatusNotFound, files.CREATED},
		{"Test UPLOADED", http.StatusNotFound, files.UPLOADED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetchMetadata := func(ctx context.Context, path string, headers filesAPISDK.Headers) (*filesAPIModels.StoredRegisteredMetaData, error) {
				return &filesAPIModels.StoredRegisteredMetaData{State: tt.state}, nil
			}

			createFileEvent := func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error) {
				t.Fatal("createFileEvent should not have been called")
				return nil, nil
			}

			downloadFile := func(path string) (io.ReadCloser, error) { return nil, nil }

			h := api.CreateV1DownloadHandler(fetchMetadata, downloadFile, createFileEvent, nil, &config.Config{})
			h.ServeHTTP(rec, req)

			assert.Equalf(t, tt.expectedStatus, rec.status, "CreateV1DownloadHandler(%v)", tt.name)
			assert.Equal(t, rec.Header().Get("Cache-Control"), "no-cache")
		})
	}
}

func TestHandleFileNotPublishedInPublishingMode(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", nil)
	req.Header.Add(dprequest.AuthHeaderKey, testAuthorizationHeader)
	req.Header.Add("X-Florence-Token", testAccessToken)

	t.Run("Test CREATED", func(t *testing.T) {
		rec := &ErrorWriter{header: make(http.Header)}

		fetchMetadata := func(ctx context.Context, path string, headers filesAPISDK.Headers) (*filesAPIModels.StoredRegisteredMetaData, error) {
			return &filesAPIModels.StoredRegisteredMetaData{State: files.CREATED}, nil
		}

		createFileEvent := func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error) {
			t.Fatal("createFileEvent should not have been called")
			return nil, nil
		}

		downloadFile := func(path string) (io.ReadCloser, error) { return FailingReadCloser{}, nil }

		h := api.CreateV1DownloadHandler(fetchMetadata, downloadFile, createFileEvent, nil, &config.Config{IsPublishing: true})
		h.ServeHTTP(rec, req)

		assert.Equalf(t, http.StatusNotFound, rec.status, "CreateV1DownloadHandler(%v)", "Test CREATED")
		assert.Equal(t, rec.Header().Get("Cache-Control"), "no-cache")
	})

	t.Run("Test UPLOADED", func(t *testing.T) {
		rec := httptest.NewRecorder()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockIdentityClient := mocks.NewMockIdentityClient(ctrl)
		mockIdentityClient.EXPECT().
			CheckTokenIdentity(gomock.Any(), testAccessToken, identity.TokenTypeUser).
			Return(&dprequest.IdentityResponse{Identifier: testIdentifier}, nil)

		fetchMetadata := func(ctx context.Context, path string, headers filesAPISDK.Headers) (*filesAPIModels.StoredRegisteredMetaData, error) {
			return &filesAPIModels.StoredRegisteredMetaData{State: files.UPLOADED}, nil
		}

		createFileEventCalled := false
		createFileEvent := func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error) {
			createFileEventCalled = true
			return nil, nil
		}

		downloadFile := func(path string) (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("testing")), nil }

		h := api.CreateV1DownloadHandler(fetchMetadata, downloadFile, createFileEvent, mockIdentityClient, &config.Config{IsPublishing: true})
		h.ServeHTTP(rec, req)

		assert.Equalf(t, http.StatusOK, rec.Code, "CreateV1DownloadHandler(%v)", "Test UPLOADED")
		assert.True(t, createFileEventCalled, "createFileEvent should have been called")
	})

	t.Run("Test UPLOADED but download fails", func(t *testing.T) {
		rec := &ErrorWriter{header: make(http.Header)}

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockIdentityClient := mocks.NewMockIdentityClient(ctrl)
		mockIdentityClient.EXPECT().
			CheckTokenIdentity(gomock.Any(), testAccessToken, identity.TokenTypeUser).
			Return(&dprequest.IdentityResponse{Identifier: testIdentifier}, nil)

		fetchMetadata := func(ctx context.Context, path string, headers filesAPISDK.Headers) (*filesAPIModels.StoredRegisteredMetaData, error) {
			return &filesAPIModels.StoredRegisteredMetaData{State: files.UPLOADED}, nil
		}

		createFileEventCalled := false
		createFileEvent := func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error) {
			createFileEventCalled = true
			return nil, nil
		}

		downloadFile := func(path string) (io.ReadCloser, error) { return nil, errors.New("error downloading file") }

		h := api.CreateV1DownloadHandler(fetchMetadata, downloadFile, createFileEvent, mockIdentityClient, &config.Config{IsPublishing: true})
		h.ServeHTTP(rec, req)

		assert.Equalf(t, http.StatusInternalServerError, rec.status, "CreateV1DownloadHandler(%v)", "Test UPLOADED but download fails")
		assert.Equal(t, rec.Header().Get("Cache-Control"), "no-cache")
		assert.True(t, createFileEventCalled, "createFileEvent should have been called")
	})
}

func TestContentTypeHeader(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", nil)
	rec := httptest.NewRecorder()

	expectedType := "text/csv"

	fetchMetadata := func(ctx context.Context, path string, headers filesAPISDK.Headers) (*filesAPIModels.StoredRegisteredMetaData, error) {
		return &filesAPIModels.StoredRegisteredMetaData{Type: expectedType, State: files.PUBLISHED}, nil
	}

	createFileEvent := func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error) {
		t.Fatal("createFileEvent should not have been called")
		return nil, nil
	}

	downloadFile := func(path string) (io.ReadCloser, error) { return FailingReadCloser{}, nil }

	h := api.CreateV1DownloadHandler(fetchMetadata, downloadFile, createFileEvent, nil, &config.Config{})
	h.ServeHTTP(rec, req)

	assert.Equal(t, expectedType, rec.Header().Get("Content-Type"))
}

func TestRedirectLocation(t *testing.T) {
	expectedUrl := "https://my-public-url.com/my-file.txt"
	var tests = []struct {
		desc         string
		publicUrlStr string
		filepath     string
	}{
		{
			desc:         "RedirectLocation correctly concatenates URL from parts with no trailing or leading slash",
			publicUrlStr: "https://my-public-url.com",
			filepath:     "my-file.txt",
		},
		{
			desc:         "RedirectLocation correctly concatenates URL with trailing slash but no leading slash",
			publicUrlStr: "https://my-public-url.com/",
			filepath:     "my-file.txt",
		},
		{
			desc:         "RedirectLocation correctly concatenates URL with leading slash but no trailing slash",
			publicUrlStr: "https://my-public-url.com",
			filepath:     "/my-file.txt",
		},
		{
			desc:         "RedirectLocation correctly concatenates URL with both trailing and leading slash",
			publicUrlStr: "https://my-public-url.com/",
			filepath:     "/my-file.txt",
		},
	}
	for _, test := range tests {
		publicUrl, _ := url.Parse(test.publicUrlStr)
		configUrl := config.ConfigUrl{*publicUrl}
		concatenatedUrl := api.RedirectLocation(&config.Config{PublicBucketURL: configUrl}, test.filepath)
		assert.Equal(t, expectedUrl, concatenatedUrl, fmt.Sprintf("testing %s: expected %s, got %s", test.desc, expectedUrl, concatenatedUrl))
	}
}
