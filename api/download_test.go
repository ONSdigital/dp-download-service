package api

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

	authMock "github.com/ONSdigital/dp-authorisation/v2/authorisation/mock"

	// "github.com/ONSdigital/dp-download-service/api"
	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/files"
	filesAPIModels "github.com/ONSdigital/dp-files-api/files"
	filesAPISDK "github.com/ONSdigital/dp-files-api/sdk"
	dprequest "github.com/ONSdigital/dp-net/v3/request"
	permissionsAPISDK "github.com/ONSdigital/dp-permissions-api/sdk"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var (
	testAccessToken         = "valid.access-token"
	testAuthorizationHeader = dprequest.BearerPrefix + testAccessToken
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

func TestHandleUnsupportedMetadataStatesWebFileMoved(t *testing.T) {
	w := httptest.NewRecorder()
	m := filesAPIModels.StoredRegisteredMetaData{
		State: files.MOVED,
	}
	publicUrl, _ := url.Parse("http://www.public-url.com")
	configUrl := config.URL{URL: *publicUrl}
	moved := handleUnsupportedMetadataStatesWeb(context.Background(), m, &config.Config{PublicBucketURL: configUrl}, "file/path", w)
	assert.Equal(t, w.Header().Get("Location"), "http://www.public-url.com/file/path")
	assert.Equal(t, moved, true)
	assert.Equal(t, w.Code, 301)
}

func TestHandleUnsupportedMetadataStatesWebFilePublished(t *testing.T) {
	w := httptest.NewRecorder()
	m := filesAPIModels.StoredRegisteredMetaData{
		State: files.PUBLISHED,
	}
	published := handleUnsupportedMetadataStatesWeb(context.Background(), m, &config.Config{}, "file/path", w)
	assert.Equal(t, published, false)
	assert.Equal(t, w.Code, 200)
}

func TestHandleUnsupportedMetadataStatesWebFileUploaded(t *testing.T) {
	w := httptest.NewRecorder()
	m := filesAPIModels.StoredRegisteredMetaData{
		State: files.UPLOADED,
	}
	uploaded := handleUnsupportedMetadataStatesWeb(context.Background(), m, &config.Config{}, "file/path", w)
	assert.Equal(t, uploaded, false)
	assert.Equal(t, w.Code, 200)
}

func TestHandlingForbiddenErrorFetchingMetadata(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", http.NoBody)
	rec := &ErrorWriter{header: make(http.Header)}

	fetchMetadata := func(ctx context.Context, path string, headers filesAPISDK.Headers) (*filesAPIModels.StoredRegisteredMetaData, error) {
		return nil, files.ErrNotAuthorised
	}

	createFileEvent := func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error) {
		t.Fatal("createFileEvent should not have been called")
		return nil, nil
	}

	downloadFile := func(path string) (io.ReadCloser, error) { return nil, nil }

	h := CreateDownloadHandlerWithAuth(fetchMetadata, downloadFile, createFileEvent, nil, &config.Config{}, nil)
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.status)
	assert.Equal(t, rec.Header().Get("Cache-Control"), "no-cache")
}

func TestHandlingNotAuthorisedErrorFetchingMetadata(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", http.NoBody)
	rec := &ErrorWriter{header: make(http.Header)}

	fetchMetadata := func(ctx context.Context, path string, headers filesAPISDK.Headers) (*filesAPIModels.StoredRegisteredMetaData, error) {
		return nil, files.ErrInvalidAuth
	}

	createFileEvent := func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error) {
		t.Fatal("createFileEvent should not have been called")
		return nil, nil
	}

	downloadFile := func(path string) (io.ReadCloser, error) { return nil, nil }

	h := CreateDownloadHandlerWithAuth(fetchMetadata, downloadFile, createFileEvent, nil, &config.Config{}, nil)
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.status)
	assert.Equal(t, rec.Header().Get("Cache-Control"), "no-cache")
}

func TestHandlingGetAuthEntityFails(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", http.NoBody)
	rec := &ErrorWriter{header: make(http.Header)}
	req.Header.Add(dprequest.AuthHeaderKey, "invalid.user-token")

	fetchMetadata := func(ctx context.Context, path string, headers filesAPISDK.Headers) (*filesAPIModels.StoredRegisteredMetaData, error) {
		return &filesAPIModels.StoredRegisteredMetaData{State: "UPLOADED"}, nil
	}

	createFileEvent := func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error) {
		t.Fatal("createFileEvent should not have been called")
		return nil, nil
	}

	authorisationMock := &authMock.MiddlewareMock{
		ParseFunc: func(token string) (*permissionsAPISDK.EntityData, error) {
			return nil, errors.New("unable to parse jwt")
		},
	}

	downloadFile := func(path string) (io.ReadCloser, error) { return nil, nil }

	h := CreateDownloadHandlerWithAuth(fetchMetadata, downloadFile, createFileEvent, authorisationMock, &config.Config{IsPublishing: true}, nil)
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.status)
	assert.Equal(t, rec.Header().Get("Cache-Control"), "no-cache")
}

func TestHandlingCheckUserPermissionsSuccess(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", http.NoBody)
	req.Header.Add(dprequest.AuthHeaderKey, "valid.user-token")

	rec := httptest.NewRecorder()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fetchMetadata := func(ctx context.Context, path string, headers filesAPISDK.Headers) (*filesAPIModels.StoredRegisteredMetaData, error) {
		return &filesAPIModels.StoredRegisteredMetaData{State: "UPLOADED", ContentItem: &filesAPIModels.StoredContentItem{DatasetID: "dataset-1", Edition: "feb-2026"}}, nil
	}

	createFileEventCalled := false
	createFileEvent := func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error) {
		createFileEventCalled = true
		return nil, nil
	}

	authorisationMock := &authMock.MiddlewareMock{
		ParseFunc: func(token string) (*permissionsAPISDK.EntityData, error) {
			return &permissionsAPISDK.EntityData{UserID: "user-1"}, nil
		},
	}

	permissionsChecker := &authMock.PermissionsCheckerMock{
		HasPermissionFunc: func(ctx context.Context, entityData permissionsAPISDK.EntityData, permission string, attributes map[string]string) (bool, error) {
			return true, nil
		},
	}
	downloadFile := func(path string) (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("testing")), nil }

	h := CreateDownloadHandlerWithAuth(fetchMetadata, downloadFile, createFileEvent, authorisationMock, &config.Config{IsPublishing: true}, permissionsChecker)
	h.ServeHTTP(rec, req)
	assert.Equalf(t, http.StatusOK, rec.Code, "CreateDownloadHandler(%v)", "Test UPLOADED")
	assert.True(t, createFileEventCalled, "createFileEvent should have been called")
}

func TestHandlingCheckUserPermissionsFails(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", http.NoBody)
	rec := &ErrorWriter{header: make(http.Header)}
	req.Header.Add(dprequest.AuthHeaderKey, "valid.user-token")

	fetchMetadata := func(ctx context.Context, path string, headers filesAPISDK.Headers) (*filesAPIModels.StoredRegisteredMetaData, error) {
		return &filesAPIModels.StoredRegisteredMetaData{State: "UPLOADED", ContentItem: &filesAPIModels.StoredContentItem{DatasetID: "dataset-1", Edition: "feb-2026"}}, nil
	}

	createFileEvent := func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error) {
		t.Fatal("createFileEvent should not have been called")
		return nil, nil
	}

	authorisationMock := &authMock.MiddlewareMock{
		ParseFunc: func(token string) (*permissionsAPISDK.EntityData, error) {
			return &permissionsAPISDK.EntityData{UserID: "user-1"}, nil
		},
	}

	permissionsChecker := &authMock.PermissionsCheckerMock{
		HasPermissionFunc: func(ctx context.Context, entityData permissionsAPISDK.EntityData, permission string, attributes map[string]string) (bool, error) {
			return false, nil
		},
	}
	downloadFile := func(path string) (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("testing")), nil }

	h := CreateDownloadHandlerWithAuth(fetchMetadata, downloadFile, createFileEvent, authorisationMock, &config.Config{IsPublishing: true}, permissionsChecker)
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.status)
}

func TestHandlingUnexpectedErrorFetchingMetadata(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", http.NoBody)
	rec := &ErrorWriter{header: make(http.Header)}

	fetchMetadata := func(ctx context.Context, path string, headers filesAPISDK.Headers) (*filesAPIModels.StoredRegisteredMetaData, error) {
		return nil, files.ErrUnknown
	}

	createFileEvent := func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error) {
		t.Fatal("createFileEvent should not have been called")
		return nil, nil
	}

	downloadFile := func(path string) (io.ReadCloser, error) { return nil, nil }

	h := CreateDownloadHandlerWithAuth(fetchMetadata, downloadFile, createFileEvent, nil, &config.Config{}, nil)
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.status)
}

func TestHandlingErrorGettingFileContent(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", http.NoBody)
	rec := &ErrorWriter{header: make(http.Header)}

	fetchMetadata := func(ctx context.Context, path string, headers filesAPISDK.Headers) (*filesAPIModels.StoredRegisteredMetaData, error) {
		return &filesAPIModels.StoredRegisteredMetaData{State: "PUBLISHED"}, nil
	}

	createFileEvent := func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error) {
		t.Fatal("createFileEvent should not have been called")
		return nil, nil
	}

	downloadFile := func(path string) (io.ReadCloser, error) { return nil, errors.New("error downloading file") }

	h := CreateDownloadHandlerWithAuth(fetchMetadata, downloadFile, createFileEvent, nil, &config.Config{}, nil)
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.status)
	assert.Equal(t, rec.Header().Get("Cache-Control"), "no-cache")
}

func TestHandlingErrorGettingFileNotAvailable(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/unavailablefile.csv", http.NoBody)
	rec := &ErrorWriter{header: make(http.Header)}

	fetchMetadata := func(ctx context.Context, path string, headers filesAPISDK.Headers) (*filesAPIModels.StoredRegisteredMetaData, error) {
		return nil, files.ErrFileNotRegistered
	}

	createFileEvent := func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error) {
		t.Fatal("createFileEvent should not have been called")
		return nil, nil
	}

	downloadFile := func(path string) (io.ReadCloser, error) { return nil, errors.New("error downloading file") }

	h := CreateDownloadHandlerWithAuth(fetchMetadata, downloadFile, createFileEvent, nil, &config.Config{}, nil)
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.status)
	assert.Equal(t, rec.Header().Get("Cache-Control"), "no-cache")
}

func TestHandleFileNotPublishedWeb(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", http.NoBody)
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
				return nil, files.ErrFileNotRegistered
			}

			downloadFile := func(path string) (io.ReadCloser, error) { return nil, nil }

			h := CreateDownloadHandlerNoAuth(fetchMetadata, downloadFile, &config.Config{})
			h.ServeHTTP(rec, req)

			assert.Equalf(t, tt.expectedStatus, rec.status, "CreateDownloadHandler(%v)", tt.name)
		})
	}
}

func TestHandleFileNotPublishedInPublishingMode(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", http.NoBody)
	req.Header.Add(dprequest.AuthHeaderKey, testAuthorizationHeader)

	permissionsChecker := &authMock.PermissionsCheckerMock{
		HasPermissionFunc: func(ctx context.Context, entityData permissionsAPISDK.EntityData, permission string, attributes map[string]string) (bool, error) {
			return true, nil
		},
	}

	authorisationMock := &authMock.MiddlewareMock{
		ParseFunc: func(token string) (*permissionsAPISDK.EntityData, error) {
			return &permissionsAPISDK.EntityData{UserID: "admin"}, nil
		},
	}

	t.Run("Test CREATED", func(t *testing.T) {
		rec := &ErrorWriter{header: make(http.Header)}

		fetchMetadata := func(ctx context.Context, path string, headers filesAPISDK.Headers) (*filesAPIModels.StoredRegisteredMetaData, error) {
			return &filesAPIModels.StoredRegisteredMetaData{State: files.CREATED}, nil
		}

		createFileEventCalled := false
		createFileEvent := func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error) {
			createFileEventCalled = true
			return nil, nil
		}

		downloadFile := func(path string) (io.ReadCloser, error) { return FailingReadCloser{}, nil }

		h := CreateDownloadHandlerWithAuth(fetchMetadata, downloadFile, createFileEvent, authorisationMock, &config.Config{IsPublishing: true}, permissionsChecker)
		h.ServeHTTP(rec, req)

		assert.Equalf(t, http.StatusNotFound, rec.status, "CreateDownloadHandler(%v)", "Test CREATED")
		assert.Equal(t, rec.Header().Get("Cache-Control"), "no-cache")
		assert.True(t, createFileEventCalled, "createFileEvent should have been called")
	})

	t.Run("Test UPLOADED", func(t *testing.T) {
		rec := httptest.NewRecorder()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fetchMetadata := func(ctx context.Context, path string, headers filesAPISDK.Headers) (*filesAPIModels.StoredRegisteredMetaData, error) {
			return &filesAPIModels.StoredRegisteredMetaData{State: files.UPLOADED}, nil
		}

		createFileEventCalled := false
		createFileEvent := func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error) {
			createFileEventCalled = true
			return nil, nil
		}

		downloadFile := func(path string) (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("testing")), nil }

		h := CreateDownloadHandlerWithAuth(fetchMetadata, downloadFile, createFileEvent, authorisationMock, &config.Config{IsPublishing: true}, permissionsChecker)
		h.ServeHTTP(rec, req)

		assert.Equalf(t, http.StatusOK, rec.Code, "CreateDownloadHandler(%v)", "Test UPLOADED")
		assert.True(t, createFileEventCalled, "createFileEvent should have been called")
	})

	t.Run("Test UPLOADED but download fails", func(t *testing.T) {
		rec := &ErrorWriter{header: make(http.Header)}

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		fetchMetadata := func(ctx context.Context, path string, headers filesAPISDK.Headers) (*filesAPIModels.StoredRegisteredMetaData, error) {
			return &filesAPIModels.StoredRegisteredMetaData{State: files.UPLOADED}, nil
		}

		createFileEventCalled := false
		createFileEvent := func(ctx context.Context, event filesAPIModels.FileEvent, headers filesAPISDK.Headers) (*filesAPIModels.FileEvent, error) {
			createFileEventCalled = true
			return nil, nil
		}

		downloadFile := func(path string) (io.ReadCloser, error) { return nil, errors.New("error downloading file") }

		h := CreateDownloadHandlerWithAuth(fetchMetadata, downloadFile, createFileEvent, authorisationMock, &config.Config{IsPublishing: true}, permissionsChecker)
		h.ServeHTTP(rec, req)

		assert.Equalf(t, http.StatusInternalServerError, rec.status, "CreateDownloadHandler(%v)", "Test UPLOADED but download fails")
		assert.Equal(t, rec.Header().Get("Cache-Control"), "no-cache")
		assert.True(t, createFileEventCalled, "createFileEvent should have been called")
	})
}

func TestContentTypeHeader(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/files/data/file.csv", http.NoBody)
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

	h := CreateDownloadHandlerWithAuth(fetchMetadata, downloadFile, createFileEvent, nil, &config.Config{}, nil)
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
		configUrl := config.URL{URL: *publicUrl}
		concatenatedUrl := RedirectLocation(&config.Config{PublicBucketURL: configUrl}, test.filepath)
		assert.Equal(t, expectedUrl, concatenatedUrl, fmt.Sprintf("testing %s: expected %s, got %s", test.desc, expectedUrl, concatenatedUrl))
	}
}
