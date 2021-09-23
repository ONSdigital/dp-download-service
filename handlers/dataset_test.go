package handlers_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ONSdigital/dp-download-service/handlers"
	"github.com/ONSdigital/dp-download-service/model"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

// failingReader is an io.Reader which always returns an error.
//
type failingReader struct{}

func (e *failingReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

// errorWithCode is an error type which wraps another error
// and can carry an http status code.
// (Kind of cheesy for lower level layers to assume caller
// has anything to do with HTTP.)
//
type errorWithCode struct {
	Event    string
	Err      error
	HTTPCode int
}

func (e *errorWithCode) Error() string {
	return e.Event
}

func (e *errorWithCode) Unwrap() error {
	return e.Err
}

func (e *errorWithCode) Code() int {
	return e.HTTPCode
}

func TestPostDataset(t *testing.T) {
	Convey("Setting up dependencies", t, func() {

		// Set up happy path clients and dependencies.
		//

		mockedModel := &ModelMock{
			CreateFunc: func(ctx context.Context, payload *model.DatasetDocument) (string, error) {
				return "fake-uuid", nil
			},
		}
		ds := handlers.NewDataset(mockedModel)

		w := httptest.NewRecorder()
		router := mux.NewRouter()
		router.Path("/downloads").Methods("POST").HandlerFunc(ds.DoPostDataset())

		Convey("happy path", func() {
			req := httptest.NewRequest(
				"POST",
				"/downloads",
				strings.NewReader(`{"downloads":{}}`),
			)
			router.ServeHTTP(w, req)

			Convey("should return created success", func() {
				So(w.Code, ShouldEqual, http.StatusCreated)
				So(w.Body.String(), ShouldContainSubstring, "uuid")
			})
		})

		Convey("when body read returns error", func() {
			req := httptest.NewRequest(
				"POST",
				"/downloads",
				&failingReader{},
			)
			router.ServeHTTP(w, req)

			Convey("handler should return error", func() {
				So(w.Code, ShouldEqual, http.StatusInternalServerError)
				So(w.Body.String(), ShouldContainSubstring, handlers.MsgCannotReadBody)
			})
		})

		Convey("when body is not valid json", func() {
			req := httptest.NewRequest(
				"POST",
				"/downloads",
				strings.NewReader(`not json`),
			)
			router.ServeHTTP(w, req)

			Convey("handler should return error", func() {
				So(w.Code, ShouldEqual, http.StatusBadRequest)
				So(w.Body.String(), ShouldContainSubstring, handlers.MsgCannotParseBody)
			})
		})

		Convey("when model returns non-code error", func() {
			mockedModel.CreateFunc = func(ctx context.Context, payload *model.DatasetDocument) (string, error) {
				return "", io.ErrUnexpectedEOF
			}

			req := httptest.NewRequest(
				"POST",
				"/downloads",
				strings.NewReader(`{"downloads":{}}`),
			)
			router.ServeHTTP(w, req)

			Convey("handler should return 500 and message", func() {
				So(w.Code, ShouldEqual, http.StatusInternalServerError)
				So(w.Body.String(), ShouldContainSubstring, handlers.MsgCannotCreateDocument)
			})
		})

		Convey("when model returns a code error", func() {
			mockedModel.CreateFunc = func(ctx context.Context, payload *model.DatasetDocument) (string, error) {
				return "", &errorWithCode{
					Event:    "create failed",
					Err:      io.ErrUnexpectedEOF,
					HTTPCode: http.StatusInsufficientStorage, // an arbitrary error for testing
				}
			}

			req := httptest.NewRequest(
				"POST",
				"/downloads",
				strings.NewReader(`{"downloads":{}}`),
			)
			router.ServeHTTP(w, req)

			Convey("handler should return code and message", func() {
				So(w.Code, ShouldEqual, http.StatusInsufficientStorage)
				So(w.Body.String(), ShouldContainSubstring, handlers.MsgCannotCreateDocument)
			})
		})
	})
}
