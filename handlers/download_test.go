package handlers

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	clientsidentity "github.com/ONSdigital/dp-api-clients-go/identity"
	"github.com/ONSdigital/dp-download-service/downloads"
	"github.com/ONSdigital/dp-download-service/handlers/mocks"
	rchttp "github.com/ONSdigital/dp-rchttp"
	"github.com/ONSdigital/go-ns/identity"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	testPublicDownload  = "http://test-public-download.com"
	testPrivateDownload = "s3://some-bucket/datasets/my-file.csv"
	testHexEncodedPSK   = "68656C6C6F20776F726C64"
	testBadEncodedPSK   = "this is not encoded"
	testPSK             = "hello world"
	testCsvContent      = "1,2,3,4"
	florenceTokenHeader = "X-Florence-Token"
	zebedeeURL          = "http://localhost:8082"
	vaultKey            = "key"
	rootVaultPath       = "/secrets/tests/psk"
)

var (
	testFilename  = "my-file.csv"
	testVaultPath = rootVaultPath + "/" + testFilename
	expectedS3Key = "/datasets/" + testFilename
	testError     = errors.New("borked")

	downloadWithPublicLink = downloads.Info{
		URL:     "/downloadURL",
		Size:    "666",
		Public:  testPublicDownload,
		Skipped: false,
	}

	downloadWithPrivateLink = downloads.Info{
		URL:     "/downloadURL",
		Size:    "666",
		Private: testPrivateDownload,
		Skipped: false,
	}

	publishedDownloadPublicLink = downloads.Model{
		IsPublished: true,
		Available:   map[string]downloads.Info{"csv": downloadWithPublicLink},
	}

	publishedDownloadPrivateLink = downloads.Model{
		IsPublished: true,
		Available:   map[string]downloads.Info{"csv": downloadWithPrivateLink},
	}

	unpublishedDownloadPrivateLink = downloads.Model{
		IsPublished: false,
		Available:   map[string]downloads.Info{"csv": downloadWithPrivateLink},
	}

	publishedDownloadNoFile = downloads.Model{
		IsPublished: true,
		Available:   map[string]downloads.Info{},
	}
)

type testClientError struct {
	code int
}

func (e testClientError) Error() string {
	return "client error"
}

func (e testClientError) Code() int {
	return e.code
}

type zeroErrReader struct {
	err error
}

func (r zeroErrReader) Read(p []byte) (int, error) {
	return copy(p, []byte{0}), r.err
}

func (r zeroErrReader) Close() error {
	return errors.New("couldn't close")
}

type errWriter struct {
	http.ResponseWriter
	err error
}

func (w errWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestDownloadDoReturnsRedirect(t *testing.T) {
	t.Parallel()
	mockCtrl := gomock.NewController(t)
	mockDownloadToken := ""
	mockServiceAuthToken := ""
	defer mockCtrl.Finish()

	Convey("Given a public link to the download exists on the filter api then return a status 301 to the download", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/filter-outputs/abcdefg.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{FilterOutputID: "abcdefg"}

		dl := mocks.NewMockDatasetDownloads(mockCtrl)
		dl.EXPECT().GetFilterOutputDownloads(gomock.Any(), params).Return(publishedDownloadPublicLink, nil)

		d := Download{DatasetDownloads: dl}

		r.HandleFunc("/downloads/filter-outputs/{filterOutputID}.csv", d.Do("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusMovedPermanently)
		So(w.Header().Get("Location"), ShouldEqual, testPublicDownload)
	})

	Convey("Given a public link to the download exists on the dataset api then return a status 301 to the download", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := mocks.NewMockDatasetDownloads(mockCtrl)
		dl.EXPECT().GetDatasetVersionDownloads(gomock.Any(), params).Return(publishedDownloadPublicLink, nil)

		d := Download{DatasetDownloads: dl}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.Do("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusMovedPermanently)
		So(w.Header().Get("Location"), ShouldEqual, testPublicDownload)
	})
}

func TestDownloadDoReturnsOK(t *testing.T) {
	t.Parallel()
	mockCtrl := gomock.NewController(t)
	mockDownloadToken := ""
	mockServiceAuthToken := ""
	defer mockCtrl.Finish()

	Convey("Given a private link to the download exists on the dataset api and the dataset is published then the file content is written to the response body", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := mocks.NewMockDatasetDownloads(mockCtrl)
		dl.EXPECT().GetDatasetVersionDownloads(gomock.Any(), params).Return(publishedDownloadPrivateLink, nil)

		s3C := mocks.NewMockS3Content(mockCtrl)
		s3C.EXPECT().
			StreamAndWrite(gomock.Any(), gomock.Eq("/datasets/my-file.csv"), gomock.Eq(w)).
			Return(nil).
			Do(func(ctx context.Context, filename string, w io.Writer) {
				w.Write([]byte(testCsvContent))
			})

		d := Download{
			DatasetDownloads: dl,
			S3Content:        s3C,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.Do("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Body.String(), ShouldEqual, testCsvContent)
	})

	Convey("Given a private link to the download exists on the dataset api and the dataset is associated but is authenticated then the file is streamed in the response body", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := mocks.NewMockDatasetDownloads(mockCtrl)
		dl.EXPECT().GetDatasetVersionDownloads(gomock.Any(), params).Return(unpublishedDownloadPrivateLink, nil)

		s3C := mocks.NewMockS3Content(mockCtrl)
		s3C.EXPECT().
			StreamAndWrite(gomock.Any(), gomock.Eq("/datasets/my-file.csv"), gomock.Eq(w)).
			Return(nil).
			Do(func(ctx context.Context, filename string, w io.Writer) {
				w.Write([]byte(testCsvContent))
			})

		d := Download{
			DatasetDownloads: dl,
			S3Content:        s3C,
			IsPublishing:     true,
		}

		httpClient := &rchttp.ClienterMock{
			DoFunc: func(ctx context.Context, req *http.Request) (*http.Response, error) {

				readCloser := ioutil.NopCloser(strings.NewReader(`{"identifier": "me"}`))

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       readCloser,
				}, nil
			},
		}
		idClient := clientsidentity.NewAPIClient(httpClient, zebedeeURL)

		chain := alice.New(identity.HandlerForHTTPClient(idClient)).Then(r)

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.Do("csv", mockServiceAuthToken, mockDownloadToken))
		req.Header.Set(florenceTokenHeader, "Florence")

		chain.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Body.String(), ShouldEqual, testCsvContent)
	})
}

/*func TestDownloadDoFailureScenarios(t *testing.T) {
	t.Parallel()
	mockCtrl := gomock.NewController(t)
	mockDownloadToken := ""
	mockServiceAuthToken := ""
	defer mockCtrl.Finish()

	Convey("Should return HTTP status not found if the dataset downloader returns a dataset version not found error", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}
		err := testClientError{http.StatusNotFound}

		dl := mocks.NewMockDatasetDownloads(mockCtrl)
		dl.EXPECT().GetDatasetVersionDownloads(gomock.Any(), params).Return(downloads.Model{}, err)

		d := Download{DatasetDownloads: dl}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.Do("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusNotFound)
		So(w.Body.String(), ShouldEqual, notFoundMessage+"\n")
	})

	Convey("Given the filter client returns an error then the download client returns this back to the caller", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/filter-outputs/abcdefg.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{FilterOutputID: "abcdefg"}
		testErr := errors.New("filter client error")

		dl := mocks.NewMockDatasetDownloads(mockCtrl)
		dl.EXPECT().GetFilterOutputDownloads(gomock.Any(), params).Return(downloads.Model{}, testErr)

		d := Download{DatasetDownloads: dl}

		r.HandleFunc("/downloads/filter-outputs/{filterOutputID}.csv", d.Do("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
		So(w.Body.String(), ShouldEqual, internalServerMessage+"\n")
	})

	Convey("Given the vault client returns an error then the download status returns an internal server error", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := mocks.NewMockDatasetDownloads(mockCtrl)
		dl.EXPECT().GetDatasetVersionDownloads(gomock.Any(), params).Return(publishedDownloadPrivateLink, nil)

		vc := mocks.NewMockVaultClient(mockCtrl)
		vc.EXPECT().ReadKey(testVaultPath, vaultKey).Return("", errors.New("vault client error"))

		d := Download{
			DatasetDownloads: dl,
			VaultClient:      vc,
			VaultPath:        rootVaultPath,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.Do("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Given the vault client returns a non hex encoded psk then an internal server error is returned by the download service", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := mocks.NewMockDatasetDownloads(mockCtrl)
		dl.EXPECT().GetDatasetVersionDownloads(gomock.Any(), params).Return(publishedDownloadPrivateLink, nil)

		vc := mocks.NewMockVaultClient(mockCtrl)
		vc.EXPECT().ReadKey(testVaultPath, vaultKey).Return(testBadEncodedPSK, nil)

		d := Download{
			DatasetDownloads: dl,
			VaultClient:      vc,
			VaultPath:        rootVaultPath,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.Do("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Given the s3 client returns an error, then an internal server error is returned by the download service", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := mocks.NewMockDatasetDownloads(mockCtrl)
		dl.EXPECT().GetDatasetVersionDownloads(gomock.Any(), params).Return(publishedDownloadPrivateLink, nil)

		vc := mocks.NewMockVaultClient(mockCtrl)
		vc.EXPECT().ReadKey(testVaultPath, vaultKey).Return(testHexEncodedPSK, nil)

		s3c := mocks.NewMockS3Client(mockCtrl)
		s3c.EXPECT().GetWithPSK(expectedS3Key, []byte(testPSK)).Return(nil, errors.New("s3 client error"))

		d := Download{
			DatasetDownloads: dl,
			VaultClient:      vc,
			S3Client:         s3c,
			VaultPath:        rootVaultPath,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.Do("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Given the s3 client cannot copy the file contents, then the download service returns an internal server error", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := mocks.NewMockDatasetDownloads(mockCtrl)
		dl.EXPECT().GetDatasetVersionDownloads(gomock.Any(), params).Return(publishedDownloadPrivateLink, nil)

		vc := mocks.NewMockVaultClient(mockCtrl)
		vc.EXPECT().ReadKey(testVaultPath, vaultKey).Return(testHexEncodedPSK, nil)

		er, ew := errors.New("readError"), errors.New("writeError")
		rdr := zeroErrReader{err: er}
		wtr := errWriter{w, ew}

		s3c := mocks.NewMockS3Client(mockCtrl)
		s3c.EXPECT().GetWithPSK(expectedS3Key, []byte(testPSK)).Return(rdr, nil)

		d := Download{
			DatasetDownloads: dl,
			VaultClient:      vc,
			S3Client:         s3c,
			VaultPath:        rootVaultPath,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.Do("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(wtr, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Given there is no file available from the dataset api then the download service returns a not found status", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := mocks.NewMockDatasetDownloads(mockCtrl)
		dl.EXPECT().GetDatasetVersionDownloads(gomock.Any(), params).Return(publishedDownloadNoFile, nil)

		d := Download{DatasetDownloads: dl}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.Do("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusNotFound)
	})
}*/
