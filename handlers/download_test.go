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

	"github.com/ONSdigital/dp-api-clients-go/v2/health"
	clientsidentity "github.com/ONSdigital/dp-api-clients-go/v2/identity"
	"github.com/ONSdigital/dp-download-service/downloads"
	"github.com/ONSdigital/dp-download-service/handlers/mocks"
	dphandlers "github.com/ONSdigital/dp-net/v2/handlers"
	dphttp "github.com/ONSdigital/dp-net/v2/http"
	"github.com/ONSdigital/dp-net/v2/request"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	testPublicDatasetDownload = "http://test-public-dataset-download.com"
	testPublicImageDownload   = "http://test-public-image-download.com"
	testPrivateCsvFilename    = "my-dataset.csv"
	testPrivateCsvS3Path      = "/datasets/my-dataset.csv"
	testPrivatePngFilename    = "my-image.png"
	testPrivatePngPath        = "/images/123/original"
	florenceTokenHeader       = "X-Florence-Token"
	testUserToken             = "UserToken"
	testServiceToken          = "ServiceToken"
	testDownloadServiceToken  = "DownloadServiceToken"
	testCollectionID          = "CollectionID"
)

var (
	testErr          = errors.New("borked")
	testCsvContent   = []byte("1,2,3,4")
	testImageContent = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
)

var (
	publishedDatasetDownloadPublicURL = downloads.Model{
		IsPublished: true,
		Public:      testPublicDatasetDownload,
	}

	publishedImageDownloadPublicURL = downloads.Model{
		IsPublished: true,
		Public:      testPublicImageDownload,
	}

	publishedDatasetDownloadPrivateURL = downloads.Model{
		IsPublished:     true,
		PrivateFilename: testPrivateCsvFilename,
		PrivateS3Path:   testPrivateCsvS3Path,
	}

	publishedImageDownloadPrivateURL = downloads.Model{
		IsPublished:     true,
		PrivateFilename: testPrivatePngFilename,
		PrivateS3Path:   testPrivatePngPath,
	}

	unpublishedDatasetDownloadPrivateLink = downloads.Model{
		IsPublished:     false,
		PrivateFilename: testPrivateCsvFilename,
		PrivateS3Path:   testPrivateCsvS3Path,
	}

	unpublishedImageDownloadPrivateLink = downloads.Model{
		IsPublished:     false,
		PrivateFilename: testPrivatePngFilename,
		PrivateS3Path:   testPrivatePngPath,
	}

	publishedDatasetDownloadNoURLs = downloads.Model{
		IsPublished: true,
	}

	publishedImageDownloadNoURLs = downloads.Model{
		IsPublished: true,
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

func TestGetDownloadParameters(t *testing.T) {
	t.Parallel()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	Convey("Given a request with UserAccess and collectionID context values", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/generic_request", nil)
		req = req.WithContext(context.WithValue(req.Context(), dphandlers.UserAccess.Context(), testUserToken))
		req = req.WithContext(context.WithValue(req.Context(), dphandlers.CollectionID.Context(), testCollectionID))

		Convey("then GetDownloadParameters extracts the values correctly", func() {
			params := GetDownloadParameters(req, testServiceToken, testDownloadServiceToken)
			So(params, ShouldResemble, downloads.Parameters{
				UserAuthToken:        testUserToken,
				ServiceAuthToken:     testServiceToken,
				DownloadServiceToken: testDownloadServiceToken,
				CollectionID:         testCollectionID,
			})
		})
	})

	Convey("Given a request without any context value", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/generic_request", nil)

		Convey("then GetDownloadParameters does not extract values from the context", func() {
			params := GetDownloadParameters(req, testServiceToken, testDownloadServiceToken)
			So(params, ShouldResemble, downloads.Parameters{
				ServiceAuthToken:     testServiceToken,
				DownloadServiceToken: testDownloadServiceToken,
			})
		})
	})
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

		dl := downloaderReturnsResult(mockCtrl, params, downloads.TypeFilterOutput, publishedDatasetDownloadPublicURL)
		s3c := s3ContentNeverInvoked(mockCtrl)

		d := Download{
			Downloader: dl,
			S3Content:  s3c,
		}

		r.HandleFunc("/downloads/filter-outputs/{filterOutputID}.csv", d.DoFilterOutput("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusMovedPermanently)
		So(w.Header().Get("Location"), ShouldEqual, testPublicDatasetDownload)
	})

	Convey("Given a public link to the download exists on the dataset api then return a status 301 to the download", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := downloaderReturnsResult(mockCtrl, params, downloads.TypeDatasetVersion, publishedDatasetDownloadPublicURL)
		s3c := s3ContentNeverInvoked(mockCtrl)

		d := Download{
			Downloader: dl,
			S3Content:  s3c,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.DoDatasetVersion("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusMovedPermanently)
		So(w.Header().Get("Location"), ShouldEqual, testPublicDatasetDownload)
	})

	Convey("Given a public link to the image download exists on the image api then return a status 301 to the download", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/images/54321/1280x720/myImage.png", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{ImageID: "54321", Variant: "1280x720", Filename: "myImage.png"}

		dl := downloaderReturnsResult(mockCtrl, params, downloads.TypeImage, publishedImageDownloadPublicURL)
		s3C := s3ContentNeverInvoked(mockCtrl)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/images/{imageID}/{variant}/{filename}", d.DoImage(mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusMovedPermanently)
		So(w.Header().Get("Location"), ShouldEqual, testPublicImageDownload)
	})
}

func TestDownloadDoReturnsOK(t *testing.T) {
	t.Parallel()
	mockCtrl := gomock.NewController(t)
	mockDownloadToken := ""
	mockServiceAuthToken := ""
	defer mockCtrl.Finish()

	Convey("Given a private link to the dataset download exists and the dataset is published then the file content is written to the response body", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := downloaderReturnsResult(mockCtrl, params, downloads.TypeDatasetVersion, publishedDatasetDownloadPrivateURL)
		s3C := s3ContentWriterSuccessfullyWritesToResponse(mockCtrl, w, testPrivateCsvS3Path, testCsvContent)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.DoDatasetVersion("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Header().Get("Content-Disposition"), ShouldEqual, "attachment; filename=my-dataset.csv")
		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Body.Bytes(), ShouldResemble, testCsvContent)
	})

	Convey("Given a private link to the dataset download exists and the dataset is associated and user is authenticated then the file is streamed in the response body", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		req = req.WithContext(context.WithValue(req.Context(), request.CallerIdentityKey, "me"))
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := downloaderReturnsResult(mockCtrl, params, downloads.TypeDatasetVersion, unpublishedDatasetDownloadPrivateLink)
		s3C := s3ContentWriterSuccessfullyWritesToResponse(mockCtrl, w, testPrivateCsvS3Path, testCsvContent)

		d := Download{
			Downloader:   dl,
			S3Content:    s3C,
			IsPublishing: true,
		}

		httpClient := &dphttp.ClienterMock{
			DoFunc: func(ctx context.Context, req *http.Request) (*http.Response, error) {

				readCloser := ioutil.NopCloser(strings.NewReader(`{"identifier": "me"}`))

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       readCloser,
				}, nil
			},
			SetPathsWithNoRetriesFunc: func(in1 []string) {},
			GetPathsWithNoRetriesFunc: func() []string { return []string{"/healthcheck"} },
		}
		hc := health.Client{Client: httpClient}
		idClient := clientsidentity.NewWithHealthClient(&hc)

		chain := alice.New(dphandlers.IdentityWithHTTPClient(idClient)).Then(r)

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.DoDatasetVersion("csv", mockServiceAuthToken, mockDownloadToken))
		req.Header.Set(florenceTokenHeader, "Florence")

		chain.ServeHTTP(w, req)

		So(w.Header().Get("Content-Disposition"), ShouldEqual, "attachment; filename=my-dataset.csv")
		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Body.Bytes(), ShouldResemble, testCsvContent)
	})

	Convey("Given a private link to the image download exists and the image is published then the file content is written to the response body", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/images/54321/1280x720/myImage.png", nil)
		req = req.WithContext(context.WithValue(req.Context(), dphandlers.UserAccess.Context(), testUserToken))
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{ImageID: "54321", Variant: "1280x720", Filename: "myImage.png", UserAuthToken: testUserToken}

		dl := downloaderReturnsResult(mockCtrl, params, downloads.TypeImage, publishedImageDownloadPrivateURL)
		s3C := s3ContentWriterSuccessfullyWritesToResponse(mockCtrl, w, testPrivatePngPath, testImageContent)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/images/{imageID}/{variant}/{filename}", d.DoImage(mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Body.Bytes(), ShouldResemble, testImageContent)
	})

	Convey("Given a private link to the image download exists and the image is not published and user is authenticated then the file is streamed in the response body", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/images/54321/1280x720/myImage.png", nil)
		req = req.WithContext(context.WithValue(req.Context(), dphandlers.UserAccess.Context(), testUserToken))
		req = req.WithContext(context.WithValue(req.Context(), request.CallerIdentityKey, "me"))
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{ImageID: "54321", Variant: "1280x720", Filename: "myImage.png", UserAuthToken: testUserToken}

		dl := downloaderReturnsResult(mockCtrl, params, downloads.TypeImage, unpublishedImageDownloadPrivateLink)
		s3C := s3ContentWriterSuccessfullyWritesToResponse(mockCtrl, w, testPrivatePngPath, testImageContent)

		d := Download{
			Downloader:   dl,
			S3Content:    s3C,
			IsPublishing: true,
		}

		httpClient := &dphttp.ClienterMock{
			DoFunc: func(ctx context.Context, req *http.Request) (*http.Response, error) {

				readCloser := ioutil.NopCloser(strings.NewReader(`{"identifier": "me"}`))

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       readCloser,
				}, nil
			},
			SetPathsWithNoRetriesFunc: func(in1 []string) {},
			GetPathsWithNoRetriesFunc: func() []string { return []string{"/healthcheck"} },
		}
		hc := health.Client{Client: httpClient}
		idClient := clientsidentity.NewWithHealthClient(&hc)

		chain := alice.New(dphandlers.IdentityWithHTTPClient(idClient)).Then(r)

		r.HandleFunc("/images/{imageID}/{variant}/{filename}", d.DoImage(mockServiceAuthToken, mockDownloadToken))
		req.Header.Set(florenceTokenHeader, "Florence")

		chain.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Body.Bytes(), ShouldResemble, testImageContent)

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

		dl := downloaderReturningError(mockCtrl, params, downloads.TypeDatasetVersion, err)
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

	Convey("Should return HTTP status not found if the image downloader returns an image not found error", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/images/54321/1280x720/myImage.png", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{ImageID: "54321", Variant: "1280x720", Filename: "myImage.png"}
		err := testClientError{http.StatusNotFound}

		dl := downloaderReturningError(mockCtrl, params, downloads.TypeImage, err)
		s3C := s3ContentNeverInvoked(mockCtrl)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/images/{imageID}/{variant}/{filename}", d.DoImage(mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusNotFound)
		So(w.Body.String(), ShouldEqual, notFoundMessage+"\n")
	})

	Convey("Should return HTTP status internal server error if the dataset downloader return an error", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := downloaderReturningError(mockCtrl, params, downloads.TypeDatasetVersion, testErr)
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

	Convey("Should return HTTP status internal server error if the image downloader return an error", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/images/54321/1280x720/myImage.png", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{ImageID: "54321", Variant: "1280x720", Filename: "myImage.png"}

		dl := downloaderReturningError(mockCtrl, params, downloads.TypeImage, testErr)
		s3C := s3ContentNeverInvoked(mockCtrl)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/images/{imageID}/{variant}/{filename}", d.DoImage(mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
		So(w.Body.String(), ShouldEqual, internalServerMessage+"\n")
	})

	Convey("Should return HTTP status internal server error if s3 content for dataset returns an unexpected error", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := downloaderReturnsResult(mockCtrl, params, downloads.TypeDatasetVersion, publishedDatasetDownloadPrivateURL)
		s3C := s3ContentReturnsAnError(mockCtrl, testErr)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.DoDatasetVersion("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
		So(w.Body.String(), ShouldEqual, internalServerMessage+"\n")
	})

	Convey("Should return HTTP status internal server error if s3 content for image returns an unexpected error", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/images/54321/1280x720/myImage.png", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{ImageID: "54321", Variant: "1280x720", Filename: "myImage.png"}

		dl := downloaderReturnsResult(mockCtrl, params, downloads.TypeImage, publishedImageDownloadPrivateURL)
		s3C := s3ContentReturnsAnError(mockCtrl, testErr)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/images/{imageID}/{variant}/{filename}", d.DoImage(mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
		So(w.Body.String(), ShouldEqual, internalServerMessage+"\n")
	})

	Convey("Should return HTTP status not found if dataset downloads has no public or private links", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := downloaderReturnsResult(mockCtrl, params, downloads.TypeDatasetVersion, publishedDatasetDownloadNoURLs)
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

	Convey("Should return HTTP status not found if image downloads has no public or private links", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/images/54321/1280x720/myImage.png", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{ImageID: "54321", Variant: "1280x720", Filename: "myImage.png"}

		dl := downloaderReturnsResult(mockCtrl, params, downloads.TypeImage, publishedImageDownloadNoURLs)
		s3C := s3ContentNeverInvoked(mockCtrl)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/images/{imageID}/{variant}/{filename}", d.DoImage(mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusNotFound)
		So(w.Body.String(), ShouldEqual, notFoundMessage+"\n")
	})
}

func downloaderReturnsResult(c *gomock.Controller, p downloads.Parameters, ft downloads.FileType, result downloads.Model) *mocks.MockDownloader {
	dl := mocks.NewMockDownloader(c)
	dl.EXPECT().Get(gomock.Any(), p, ft, gomock.Any()).Return(result, nil)
	return dl
}

func downloaderReturningError(c *gomock.Controller, p downloads.Parameters, ft downloads.FileType, err error) *mocks.MockDownloader {
	dl := mocks.NewMockDownloader(c)
	dl.EXPECT().Get(gomock.Any(), p, ft, gomock.Any()).Return(downloads.Model{}, err)
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

func s3ContentWriterSuccessfullyWritesToResponse(c *gomock.Controller, w io.Writer, expectedS3Path string, expectedBody []byte) *mocks.MockS3Content {
	s3C := mocks.NewMockS3Content(c)
	s3C.EXPECT().
		StreamAndWrite(gomock.Any(), gomock.Eq(expectedS3Path), gomock.Eq(w)).
		Return(nil).
		Do(func(ctx context.Context, expectedS3Path string, w io.Writer) {
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
