package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	clientsidentity "github.com/ONSdigital/dp-api-clients-go/identity"
	"github.com/ONSdigital/dp-download-service/downloads"
	"github.com/ONSdigital/dp-download-service/handlers/mocks"
	dpnethandlers "github.com/ONSdigital/dp-net/handlers"
	rchttp "github.com/ONSdigital/dp-rchttp"
	"github.com/ONSdigital/go-ns/identity"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	testPublicDatasetDownload = "http://test-public-dataset-download.com"
	testPublicImageDownload   = "http://test-public-image-download.com"
	testPrivateDownloadFmt    = "s3://some-bucket%s"
	testPrivateCsvS3Key       = "/datasets/my-dataset.csv"
	testPrivatePngS3Key       = "/datasets/my-image.png"
	florenceTokenHeader       = "X-Florence-Token"
	zebedeeURL                = "http://localhost:8082"
	testUserToken             = "UserToken"
	testServiceToken          = "ServiceToken"
	testDownloadServiceToken  = "DownloadServiceToken"
	testCollectionID          = "CollectionID"
)

var (
	testError        = errors.New("borked")
	testCsvContent   = []byte("1,2,3,4")
	testImageContent = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
)

// generate download Info with provided public URL
func infoWithPublicURL(publicDownload string) downloads.Info {
	return downloads.Info{
		URL:     "/downloadURL",
		Size:    "666",
		Public:  publicDownload,
		Skipped: false,
	}
}

// generate download Info with provided private S3 key
func infoWithPrivateURL(privateS3Key string) downloads.Info {
	return downloads.Info{
		URL:     "/downloadURL",
		Size:    "666",
		Private: fmt.Sprintf(testPrivateDownloadFmt, privateS3Key),
		Skipped: false,
	}
}

// generate download Info with no URL
func infoWithNoURLs() downloads.Info {
	return downloads.Info{
		URL:     "/downloadURL",
		Size:    "666",
		Skipped: false,
	}
}

// generate download Info with an invalid private URL
func infoWithInvalidPrivateURL() downloads.Info {
	return downloads.Info{
		URL:     "/downloadURL",
		Size:    "666",
		Skipped: false,
		Private: "@£$%^&*()_+",
	}
}

var (
	publishedDatasetDownloadPublicURL = downloads.Model{
		IsPublished: true,
		Available:   map[string]map[string]downloads.Info{"csv": {downloads.VariantDefault: infoWithPublicURL(testPublicDatasetDownload)}},
	}

	publishedImageDownloadPublicURL = downloads.Model{
		IsPublished: true,
		Available:   map[string]map[string]downloads.Info{"png": {"1280x720": infoWithPublicURL(testPublicImageDownload)}},
	}

	publishedDatasetDownloadPrivateURL = downloads.Model{
		IsPublished: true,
		Available:   map[string]map[string]downloads.Info{"csv": {downloads.VariantDefault: infoWithPrivateURL(testPrivateCsvS3Key)}},
	}

	publishedImageDownloadPrivateURL = downloads.Model{
		IsPublished: true,
		Available:   map[string]map[string]downloads.Info{"png": {"1280x720": infoWithPrivateURL(testPrivatePngS3Key)}},
	}

	unpublishedDatasetDownloadPrivateLink = downloads.Model{
		IsPublished: false,
		Available:   map[string]map[string]downloads.Info{"csv": {downloads.VariantDefault: infoWithPrivateURL(testPrivateCsvS3Key)}},
	}

	unpublishedImageDownloadPrivateLink = downloads.Model{
		IsPublished: false,
		Available:   map[string]map[string]downloads.Info{"png": {"1280x720": infoWithPrivateURL(testPrivatePngS3Key)}},
	}

	publishedDatasetDownloadNoURLs = downloads.Model{
		IsPublished: true,
		Available:   map[string]map[string]downloads.Info{"csv": {downloads.VariantDefault: infoWithNoURLs()}},
	}

	publishedImageDownloadNoURLs = downloads.Model{
		IsPublished: true,
		Available:   map[string]map[string]downloads.Info{"png": {"1280x720": infoWithNoURLs()}},
	}

	publishedDatasetDownloadInvalidPrivateURL = downloads.Model{
		IsPublished: true,
		Available:   map[string]map[string]downloads.Info{"csv": {downloads.VariantDefault: infoWithInvalidPrivateURL()}},
	}

	publishedImageDownloadInvalidPrivateURL = downloads.Model{
		IsPublished: true,
		Available:   map[string]map[string]downloads.Info{"png": {"1280x720": infoWithInvalidPrivateURL()}},
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
		req = req.WithContext(context.WithValue(req.Context(), dpnethandlers.UserAccess.Context(), testUserToken))
		req = req.WithContext(context.WithValue(req.Context(), dpnethandlers.CollectionID.Context(), testCollectionID))

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

		params := downloads.Parameters{ImageID: "54321", Variant: "1280x720", Name: "myImage", Ext: "png"}

		dl := downloaderReturnsResult(mockCtrl, params, downloads.TypeImage, publishedImageDownloadPublicURL)
		s3C := s3ContentNeverInvoked(mockCtrl)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/images/{imageID}/{variant}/{name}.{ext}", d.DoImage(mockServiceAuthToken, mockDownloadToken))
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
		s3C := s3ContentWriterSuccessfullyWritesToResponse(mockCtrl, w, testPrivateCsvS3Key, testCsvContent)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.DoDatasetVersion("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Body.Bytes(), ShouldResemble, testCsvContent)
	})

	Convey("Given a private link to the dataset download exists and the dataset is published then the file content is written to the response body", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := downloaderReturnsResult(mockCtrl, params, downloads.TypeDatasetVersion, publishedDatasetDownloadPrivateURL)
		s3C := s3ContentWriterSuccessfullyWritesToResponse(mockCtrl, w, testPrivateCsvS3Key, testCsvContent)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.DoDatasetVersion("csv", mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Body.Bytes(), ShouldResemble, testCsvContent)
	})

	Convey("Given a private link to the dataset download exists and the dataset is associated and user is authenticated then the file is streamed in the response body", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := downloaderReturnsResult(mockCtrl, params, downloads.TypeDatasetVersion, unpublishedDatasetDownloadPrivateLink)
		s3C := s3ContentWriterSuccessfullyWritesToResponse(mockCtrl, w, testPrivateCsvS3Key, testCsvContent)

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

	Convey("Given a private link to the image download exists and the image is published then the file content is written to the response body", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/images/54321/1280x720/myImage.png", nil)
		req = req.WithContext(context.WithValue(req.Context(), dpnethandlers.UserAccess.Context(), testUserToken))
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{ImageID: "54321", Variant: "1280x720", Name: "myImage", Ext: "png", UserAuthToken: testUserToken}

		dl := downloaderReturnsResult(mockCtrl, params, downloads.TypeImage, publishedImageDownloadPrivateURL)
		s3C := s3ContentWriterSuccessfullyWritesToResponse(mockCtrl, w, testPrivatePngS3Key, testImageContent)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/images/{imageID}/{variant}/{name}.{ext}", d.DoImage(mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Body.Bytes(), ShouldResemble, testImageContent)
	})

	Convey("Given a private link to the image download exists and the image is not published and user is authenticated then the file is streamed in the response body", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/images/54321/1280x720/myImage.png", nil)
		req = req.WithContext(context.WithValue(req.Context(), dpnethandlers.UserAccess.Context(), testUserToken))
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{ImageID: "54321", Variant: "1280x720", Name: "myImage", Ext: "png", UserAuthToken: testUserToken}

		dl := downloaderReturnsResult(mockCtrl, params, downloads.TypeImage, unpublishedImageDownloadPrivateLink)
		s3C := s3ContentWriterSuccessfullyWritesToResponse(mockCtrl, w, testPrivatePngS3Key, testImageContent)

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

		r.HandleFunc("/images/{imageID}/{variant}/{name}.{ext}", d.DoImage(mockServiceAuthToken, mockDownloadToken))
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

		params := downloads.Parameters{ImageID: "54321", Variant: "1280x720", Name: "myImage", Ext: "png"}
		err := testClientError{http.StatusNotFound}

		dl := downloaderReturningError(mockCtrl, params, downloads.TypeImage, err)
		s3C := s3ContentNeverInvoked(mockCtrl)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/images/{imageID}/{variant}/{name}.{ext}", d.DoImage(mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusNotFound)
		So(w.Body.String(), ShouldEqual, notFoundMessage+"\n")
	})

	Convey("Should return HTTP status internal server error if the dataset downloader return an error", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := downloaderReturningError(mockCtrl, params, downloads.TypeDatasetVersion, testError)
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

		params := downloads.Parameters{ImageID: "54321", Variant: "1280x720", Name: "myImage", Ext: "png"}

		dl := downloaderReturningError(mockCtrl, params, downloads.TypeImage, testError)
		s3C := s3ContentNeverInvoked(mockCtrl)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/images/{imageID}/{variant}/{name}.{ext}", d.DoImage(mockServiceAuthToken, mockDownloadToken))
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

	Convey("Should return HTTP status internal server error if s3 content for image returns an unexpected error", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/images/54321/1280x720/myImage.png", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{ImageID: "54321", Variant: "1280x720", Name: "myImage", Ext: "png"}

		dl := downloaderReturnsResult(mockCtrl, params, downloads.TypeImage, publishedImageDownloadPrivateURL)
		s3C := s3ContentReturnsAnError(mockCtrl, testError)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/images/{imageID}/{variant}/{name}.{ext}", d.DoImage(mockServiceAuthToken, mockDownloadToken))
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

		params := downloads.Parameters{ImageID: "54321", Variant: "1280x720", Name: "myImage", Ext: "png"}

		dl := downloaderReturnsResult(mockCtrl, params, downloads.TypeImage, publishedImageDownloadNoURLs)
		s3C := s3ContentNeverInvoked(mockCtrl)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/images/{imageID}/{variant}/{name}.{ext}", d.DoImage(mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusNotFound)
		So(w.Body.String(), ShouldEqual, notFoundMessage+"\n")
	})

	Convey("Should return HTTP status internal server error if dataset downloads has an an invalid private URL", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{DatasetID: "12345", Edition: "6789", Version: "1"}

		dl := downloaderReturnsResult(mockCtrl, params, downloads.TypeDatasetVersion, publishedDatasetDownloadInvalidPrivateURL)
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

	Convey("Should return HTTP status internal server error if image downloads has an an invalid private URL", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/images/54321/1280x720/myImage.png", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		params := downloads.Parameters{ImageID: "54321", Variant: "1280x720", Name: "myImage", Ext: "png"}

		dl := downloaderReturnsResult(mockCtrl, params, downloads.TypeImage, publishedImageDownloadInvalidPrivateURL)
		s3C := s3ContentNeverInvoked(mockCtrl)

		d := Download{
			Downloader: dl,
			S3Content:  s3C,
		}

		r.HandleFunc("/images/{imageID}/{variant}/{name}.{ext}", d.DoImage(mockServiceAuthToken, mockDownloadToken))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
		So(w.Body.String(), ShouldEqual, internalServerMessage+"\n")
	})
}

func downloaderReturnsResult(c *gomock.Controller, p downloads.Parameters, ft downloads.FileType, result downloads.Model) *mocks.MockDownloader {
	dl := mocks.NewMockDownloader(c)
	dl.EXPECT().Get(gomock.Any(), p, ft).Return(result, nil)
	return dl
}

func downloaderReturningError(c *gomock.Controller, p downloads.Parameters, ft downloads.FileType, err error) *mocks.MockDownloader {
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
		StreamAndWrite(gomock.Any(), gomock.Eq(expectedFilename), gomock.Eq(w)).
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
