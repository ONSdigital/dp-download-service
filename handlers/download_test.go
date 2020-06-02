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
	florenceTokenHeader = "X-Florence-Token"
	zebedeeURL          = "http://localhost:8082"
)

var (
	testCsvContent   = []byte("1,2,3,4")
	testImageContent = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
)

var (
	testError = errors.New("borked")

	downloadWithPublicURL = downloads.Info{
		URL:     "/downloadURL",
		Size:    "666",
		Public:  testPublicDownload,
		Skipped: false,
	}

	downloadWithPrivateURL = downloads.Info{
		URL:     "/downloadURL",
		Size:    "666",
		Private: testPrivateDownload,
		Skipped: false,
	}

	downloadWithNoURLs = downloads.Info{
		URL:     "/downloadURL",
		Size:    "666",
		Skipped: false,
	}

	downloadWithInvalidPrivateURL = downloads.Info{
		URL:     "/downloadURL",
		Size:    "666",
		Skipped: false,
		Private: "@£$%^&*()_+",
	}

	publishedDownloadPublicURL = downloads.Model{
		IsPublished: true,
		Available:   map[string]map[string]downloads.Info{"csv": {downloads.VariantDefault: downloadWithPublicURL}},
	}

	publishedDownloadPrivateURL = downloads.Model{
		IsPublished: true,
		Available:   map[string]map[string]downloads.Info{"csv": {downloads.VariantDefault: downloadWithPrivateURL}},
	}

	unpublishedDownloadPrivateLink = downloads.Model{
		IsPublished: false,
		Available:   map[string]map[string]downloads.Info{"csv": {downloads.VariantDefault: downloadWithPrivateURL}},
	}

	publishedDownloadNoURLs = downloads.Model{
		IsPublished: true,
		Available:   map[string]map[string]downloads.Info{"csv": {downloads.VariantDefault: downloadWithNoURLs}},
	}

	publishedDownloadInvalidPrivateURL = downloads.Model{
		IsPublished: true,
		Available:   map[string]map[string]downloads.Info{"csv": {downloads.VariantDefault: downloadWithInvalidPrivateURL}},
	}

	publishedImageURL = downloads.Model{
		IsPublished: true,
		Available:   map[string]map[string]downloads.Info{"png": {"1280x720": downloadWithPublicURL}},
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

		dl := datasetDownloadsReturnsResult(mockCtrl, params, downloads.TypeFilterOutput, publishedDownloadPublicURL)
		s3c := s3ContentNeverInvoked(mockCtrl)

		d := Download{
			Downloader: dl,
			S3Content:  s3c,
		}

		r.HandleFunc("/downloads/filter-outputs/{filterOutputID}.csv", d.DoFilterOutput("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusMovedPermanently)
		So(w.Header().Get("Location"), ShouldEqual, testPublicDownload)
	})

	Convey("Given a public link to the download exists on the dataset api then return a status 301 to the download", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := datasetDownloadsReturnsResult(mockCtrl, params, downloads.TypeDatasetVersion, publishedDownloadPublicURL)
		s3c := s3ContentNeverInvoked(mockCtrl)

		d := Download{
			Downloader: dl,
			S3Content:  s3c,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.DoDatasetVersion("csv", mockServiceAuthToken, mockDownloadToken))
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

	Convey("Given a private link to the download exists and the dataset is published then the file content is written to the response body", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := datasetDownloadsReturnsResult(mockCtrl, params, downloads.TypeDatasetVersion, publishedDownloadPrivateURL)
		s3C := s3ContentWriterSuccessfullyWritesToResponse(mockCtrl, w, "/datasets/my-file.csv", testCsvContent)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.DoDatasetVersion("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Body.Bytes(), ShouldResemble, testCsvContent)
	})

	Convey("Given a private link to the download exists and the dataset is associated but is authenticated then the file is streamed in the response body", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := datasetDownloadsReturnsResult(mockCtrl, params, downloads.TypeDatasetVersion, unpublishedDownloadPrivateLink)
		s3C := s3ContentWriterSuccessfullyWritesToResponse(mockCtrl, w, "/datasets/my-file.csv", testCsvContent)

		d := Download{
			Downloader:   dl,
			S3Content:    s3C,
			IsPublishing: true,
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

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.DoDatasetVersion("csv", mockServiceAuthToken, mockDownloadToken))
		req.Header.Set(florenceTokenHeader, "Florence")

		chain.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Body.Bytes(), ShouldResemble, testCsvContent)
	})
}

func TestDownloadDoFailureScenarios(t *testing.T) {
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

		dl := datasetDownloadsReturningError(mockCtrl, params, downloads.TypeDatasetVersion, err)
		s3C := s3ContentNeverInvoked(mockCtrl)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.DoDatasetVersion("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusNotFound)
		So(w.Body.String(), ShouldEqual, notFoundMessage+"\n")
	})

	Convey("Should return HTTP status internal server error if the dataset downloader return an error", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := datasetDownloadsReturningError(mockCtrl, params, downloads.TypeDatasetVersion, testError)
		s3C := s3ContentNeverInvoked(mockCtrl)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.DoDatasetVersion("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
		So(w.Body.String(), ShouldEqual, internalServerMessage+"\n")
	})

	Convey("Should return HTTP status internal server error if s3 content returns an unexpected error", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := datasetDownloadsReturnsResult(mockCtrl, params, downloads.TypeDatasetVersion, publishedDownloadPrivateURL)
		s3C := s3ContentReturnsAnError(mockCtrl, testError)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.DoDatasetVersion("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
		So(w.Body.String(), ShouldEqual, internalServerMessage+"\n")
	})

	Convey("Should return HTTP status not found if dataset downloads has no public or private links", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := datasetDownloadsReturnsResult(mockCtrl, params, downloads.TypeDatasetVersion, publishedDownloadNoURLs)
		s3C := s3ContentNeverInvoked(mockCtrl)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.DoDatasetVersion("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusNotFound)
		So(w.Body.String(), ShouldEqual, notFoundMessage+"\n")
	})

	Convey("Should return HTTP status internal server error if dataset downloads has an an invalid private URL", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := datasetDownloadsReturnsResult(mockCtrl, params, downloads.TypeDatasetVersion, publishedDownloadInvalidPrivateURL)
		s3C := s3ContentNeverInvoked(mockCtrl)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.DoDatasetVersion("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
		So(w.Body.String(), ShouldEqual, internalServerMessage+"\n")
	})
}

func TestDownloadImage(t *testing.T) {
	t.Parallel()
	mockCtrl := gomock.NewController(t)
	mockDownloadToken := ""
	mockServiceAuthToken := ""
	defer mockCtrl.Finish()

	// TODO create more test cases for DownloadImage

	Convey("Image endpoint accessible", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/images/54321/1280x720/myImage.png", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{ImageID: "54321", Variant: "1280x720", Name: "myImage", Ext: "png"}

		dl := datasetDownloadsReturnsResult(mockCtrl, params, downloads.TypeImage, publishedImageURL)
		// s3C := s3ContentWriterSuccessfullyWritesToResponse(mockCtrl, w, "/datasets/my-file.csv", testImageContent)
		s3C := s3ContentWriterSuccessfullyWritesToResponse(mockCtrl, w, "myImage.png", testImageContent)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/images/{imageID}/{variant}/{name}.{ext}", d.DoImage(mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusMovedPermanently)
		So(w.Header().Get("Location"), ShouldEqual, testPublicDownload)

		// So(w.Code, ShouldEqual, http.StatusOK)
		// So(w.Body.Bytes(), ShouldResemble, testImageContent)
	})
}

func datasetDownloadsReturnsResult(c *gomock.Controller, p downloads.Parameters, ft downloads.FileType, result downloads.Model) *mocks.MockDownloader {
	dl := mocks.NewMockDownloader(c)
	dl.EXPECT().Get(gomock.Any(), p, ft).Return(result, nil)
	return dl
}

func datasetDownloadsReturningError(c *gomock.Controller, p downloads.Parameters, ft downloads.FileType, err error) *mocks.MockDownloader {
	dl := mocks.NewMockDownloader(c)
	dl.EXPECT().Get(gomock.Any(), p, ft).Return(downloads.Model{}, err)
	return dl
}

func s3ContentNeverInvoked(c *gomock.Controller) *mocks.MockS3Content {
	s3C := mocks.NewMockS3Content(c)
	s3C.EXPECT().
		StreamAndWrite(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0).
		Return(nil)
	return s3C
}

func s3ContentWriterSuccessfullyWritesToResponse(c *gomock.Controller, w io.Writer, expectedFilename string, expectedBody []byte) *mocks.MockS3Content {
	s3C := mocks.NewMockS3Content(c)
	s3C.EXPECT().
		// StreamAndWrite(gomock.Any(), gomock.Eq(expectedFilename), gomock.Eq(w)).
		StreamAndWrite(gomock.Any(), gomock.Any(), gomock.Eq(w)).
		Return(nil).
		Do(func(ctx context.Context, filename string, w io.Writer) {
			w.Write(expectedBody)
		})
	return s3C
}

func s3ContentReturnsAnError(c *gomock.Controller, err error) *mocks.MockS3Content {
	s3C := mocks.NewMockS3Content(c)
	s3C.EXPECT().
		StreamAndWrite(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(err)

	return s3C
}
