package handlers

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ONSdigital/go-ns/identity"
	"github.com/justinas/alice"

	"github.com/ONSdigital/dp-download-service/handlers/mocks"
	"github.com/ONSdigital/go-ns/clients/dataset"
	"github.com/ONSdigital/go-ns/clients/filter"
	"github.com/ONSdigital/go-ns/identity/identitytest"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/golang/mock/gomock"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

type testContextKey string

const (
	testPublicDownload  = "http://test-public-download.com"
	testPrivateDownload = "s3://some-bucket/my-file.csv"
	testHexEncodedPSK   = "68656C6C6F20776F726C64"
	testBadEncodedPSK   = "this is not encoded"
	testPSK             = "hello world"
	testVaultPath       = "/secrets/tests/psk"
	testCsvContent      = "1,2,3,4"
	testSecretKey       = "shhh it's a secret"
	florenceTokenHeader = "X-Florence-Token"
	testUserContext     = testContextKey("User-Identity")
)

var (
	testBucket   = "some-bucket"
	testFilename = "my-file.csv"
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
	defer mockCtrl.Finish()

	Convey("Given a public link to the download exists on the filter api then return a status 301 to the download", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/filter-outputs/abcdefg.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		fc := mocks.NewMockFilterClient(mockCtrl)
		fo := filter.Model{
			Downloads: map[string]filter.Download{
				"csv": {
					Public: testPublicDownload,
				},
			},
			IsPublished: true,
		}
		fc.EXPECT().GetOutput(gomock.Any(), "abcdefg").Return(fo, nil)
		d := Download{
			FilterClient: fc,
		}

		r.HandleFunc("/downloads/filter-outputs/{filterOutputID}.csv", d.Do("csv"))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusMovedPermanently)
		So(w.Header().Get("Location"), ShouldEqual, testPublicDownload)
	})

	Convey("Given a public link to the download exists on the dataset api then return a status 301 to the download", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		dc := mocks.NewMockDatasetClient(mockCtrl)
		ver := dataset.Version{
			Downloads: map[string]dataset.Download{
				"csv": {
					Public: testPublicDownload,
				},
			},
			State: "published",
		}
		dc.EXPECT().GetVersion("12345", "6789", "1", gomock.Any()).Return(ver, nil)
		d := Download{
			DatasetClient: dc,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.Do("csv"))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusMovedPermanently)
		So(w.Header().Get("Location"), ShouldEqual, testPublicDownload)
	})
}

func TestDownloadDoReturnsOK(t *testing.T) {
	t.Parallel()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	Convey("Given a private link to the download exists on the dataset api and the dataset is published then the file is streamed in the response body", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		dc := mocks.NewMockDatasetClient(mockCtrl)
		ver := dataset.Version{
			Downloads: map[string]dataset.Download{
				"csv": {
					Private: testPrivateDownload,
				},
			},
			State: "published",
		}
		dc.EXPECT().GetVersion("12345", "6789", "1", gomock.Any()).Return(ver, nil)

		vc := mocks.NewMockVaultClient(mockCtrl)
		vc.EXPECT().ReadKey(testVaultPath, testFilename).Return(testHexEncodedPSK, nil)

		input := &s3.GetObjectInput{
			Bucket: &testBucket,
			Key:    &testFilename,
		}

		output := &s3.GetObjectOutput{
			Body: ioutil.NopCloser(strings.NewReader(testCsvContent)),
		}
		s3c := mocks.NewMockS3Client(mockCtrl)
		s3c.EXPECT().GetObjectWithPSK(input, []byte(testPSK)).Return(output, nil)

		d := Download{
			DatasetClient: dc,
			VaultClient:   vc,
			S3Client:      s3c,
			BucketName:    testBucket,
			VaultPath:     testVaultPath,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.Do("csv"))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Body.String(), ShouldEqual, testCsvContent)
	})

	Convey("Given a private link to the download exists on the dataset api and the dataset is associated but is authenticated then the file is streamed in the response body", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		dc := mocks.NewMockDatasetClient(mockCtrl)
		ver := dataset.Version{
			Downloads: map[string]dataset.Download{
				"csv": {
					Private: testPrivateDownload,
				},
			},
			State: "associated",
		}
		dc.EXPECT().GetVersion("12345", "6789", "1", gomock.Any()).Return(ver, nil)

		vc := mocks.NewMockVaultClient(mockCtrl)
		vc.EXPECT().ReadKey(testVaultPath, testFilename).Return(testHexEncodedPSK, nil)

		input := &s3.GetObjectInput{
			Bucket: &testBucket,
			Key:    &testFilename,
		}

		output := &s3.GetObjectOutput{
			Body: ioutil.NopCloser(strings.NewReader(testCsvContent)),
		}
		s3c := mocks.NewMockS3Client(mockCtrl)
		s3c.EXPECT().GetObjectWithPSK(input, []byte(testPSK)).Return(output, nil)

		d := Download{
			DatasetClient: dc,
			VaultClient:   vc,
			S3Client:      s3c,
			BucketName:    testBucket,
			VaultPath:     testVaultPath,
			SecretKey:     testSecretKey,
			IsPublishing:  true,
		}

		httpClient := &identitytest.HTTPClientMock{
			DoFunc: func(ctx context.Context, req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(strings.NewReader(`{"identifier": "me"}`)),
				}, nil
			},
		}

		chain := alice.New(identity.HandlerForHTTPClient(true, httpClient, "")).Then(r)

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.Do("csv"))
		req.Header.Set(florenceTokenHeader, "Florence")

		chain.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusOK)
		So(w.Body.String(), ShouldEqual, testCsvContent)
	})
}

func TestDownloadDoFailureScenarios(t *testing.T) {
	t.Parallel()
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	Convey("Given the dataset client returns a status not found then the download client returns this status back to the caller", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		dc := mocks.NewMockDatasetClient(mockCtrl)
		err := testClientError{http.StatusNotFound}
		dc.EXPECT().GetVersion("12345", "6789", "1", gomock.Any()).Return(dataset.Version{}, err)
		d := Download{
			DatasetClient: dc,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.Do("csv"))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusNotFound)
		So(w.Body.String(), ShouldEqual, notFoundMessage+"\n")
	})

	Convey("Given the filter client returns an error then the download client returns this back to the caller", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/filter-outputs/abcdefg.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		fc := mocks.NewMockFilterClient(mockCtrl)
		testErr := errors.New("filter client error")
		fc.EXPECT().GetOutput(gomock.Any(), "abcdefg").Return(filter.Model{}, testErr)
		d := Download{
			FilterClient: fc,
		}

		r.HandleFunc("/downloads/filter-outputs/{filterOutputID}.csv", d.Do("csv"))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
		So(w.Body.String(), ShouldEqual, internalServerMessage+"\n")
	})

	Convey("Given the vault client returns an error then the download status returns an internal server error", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		dc := mocks.NewMockDatasetClient(mockCtrl)
		ver := dataset.Version{
			Downloads: map[string]dataset.Download{
				"csv": {
					Private: testPrivateDownload,
				},
			},
			State: "published",
		}
		dc.EXPECT().GetVersion("12345", "6789", "1", gomock.Any()).Return(ver, nil)

		vc := mocks.NewMockVaultClient(mockCtrl)
		vc.EXPECT().ReadKey(testVaultPath, testFilename).Return("", errors.New("vault client error"))

		d := Download{
			DatasetClient: dc,
			VaultClient:   vc,
			VaultPath:     testVaultPath,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.Do("csv"))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Given the vault client returns a non hex encoded psk then an internal server error is returned by the download service", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		dc := mocks.NewMockDatasetClient(mockCtrl)
		ver := dataset.Version{
			Downloads: map[string]dataset.Download{
				"csv": {
					Private: testPrivateDownload,
				},
			},
			State: "published",
		}
		dc.EXPECT().GetVersion("12345", "6789", "1", gomock.Any()).Return(ver, nil)

		vc := mocks.NewMockVaultClient(mockCtrl)
		vc.EXPECT().ReadKey(testVaultPath, testFilename).Return(testBadEncodedPSK, nil)

		d := Download{
			DatasetClient: dc,
			VaultClient:   vc,
			VaultPath:     testVaultPath,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.Do("csv"))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Given the s3 client returns an error, then an internal server error is returned by the download service", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		dc := mocks.NewMockDatasetClient(mockCtrl)
		ver := dataset.Version{
			Downloads: map[string]dataset.Download{
				"csv": {
					Private: testPrivateDownload,
				},
			},
			State: "published",
		}
		dc.EXPECT().GetVersion("12345", "6789", "1", gomock.Any()).Return(ver, nil)

		vc := mocks.NewMockVaultClient(mockCtrl)
		vc.EXPECT().ReadKey(testVaultPath, testFilename).Return(testHexEncodedPSK, nil)

		input := &s3.GetObjectInput{
			Bucket: &testBucket,
			Key:    &testFilename,
		}

		s3c := mocks.NewMockS3Client(mockCtrl)
		s3c.EXPECT().GetObjectWithPSK(input, []byte(testPSK)).Return(nil, errors.New("s3 client error"))

		d := Download{
			DatasetClient: dc,
			VaultClient:   vc,
			S3Client:      s3c,
			BucketName:    testBucket,
			VaultPath:     testVaultPath,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.Do("csv"))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Given the s3 client cannot copy the file contents, then the download service returns an internal server error", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		dc := mocks.NewMockDatasetClient(mockCtrl)
		ver := dataset.Version{
			Downloads: map[string]dataset.Download{
				"csv": {
					Private: testPrivateDownload,
				},
			},
			State: "published",
		}
		dc.EXPECT().GetVersion("12345", "6789", "1", gomock.Any()).Return(ver, nil)

		vc := mocks.NewMockVaultClient(mockCtrl)
		vc.EXPECT().ReadKey(testVaultPath, testFilename).Return(testHexEncodedPSK, nil)

		input := &s3.GetObjectInput{
			Bucket: &testBucket,
			Key:    &testFilename,
		}

		er, ew := errors.New("readError"), errors.New("writeError")
		rdr := zeroErrReader{err: er}
		wtr := errWriter{w, ew}

		output := &s3.GetObjectOutput{
			Body: rdr,
		}
		s3c := mocks.NewMockS3Client(mockCtrl)
		s3c.EXPECT().GetObjectWithPSK(input, []byte(testPSK)).Return(output, nil)

		d := Download{
			DatasetClient: dc,
			VaultClient:   vc,
			S3Client:      s3c,
			BucketName:    testBucket,
			VaultPath:     testVaultPath,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.Do("csv"))
		r.ServeHTTP(wtr, req)

		So(w.Code, ShouldEqual, http.StatusInternalServerError)
	})

	Convey("Given there is no file available from the dataset api then the download service returns a not found status", t, func() {
		req := httptest.NewRequest("GET", "http://localhost:28000/downloads/datasets/12345/editions/6789/versions/1.csv", nil)
		w := httptest.NewRecorder()
		r := mux.NewRouter()

		dc := mocks.NewMockDatasetClient(mockCtrl)
		dc.EXPECT().GetVersion("12345", "6789", "1", gomock.Any()).Return(dataset.Version{}, nil)
		d := Download{
			DatasetClient: dc,
		}

		r.HandleFunc("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv", d.Do("csv"))
		r.ServeHTTP(w, req)

		So(w.Code, ShouldEqual, http.StatusNotFound)
	})
}
