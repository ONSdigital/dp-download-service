package handlers

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ONSdigital/log.go/v2/log"

	. "github.com/smartystreets/goconvey/convey"
)

type testError struct {
	err     error
	logData map[string]interface{}
}

// Error implements the Go standard error interface
func (e *testError) Error() string {
	if e.err == nil {
		return "nil"
	}
	return e.err.Error()
}

// LogData implements the DataLogger interface which allows you extract
// embedded log.Data from an error
func (e *testError) LogData() map[string]interface{} {
	if e.logData == nil {
		return log.Data{}
	}
	return e.logData
}

// Unwrap allows unwrapping an error to access the underlying error
func (e *testError) Unwrap() error {
	return e.err
}

func TestCallbackHappy(t *testing.T) {

	Convey("Given an error with embedded logData", t, func() {
		err := &testError{
			logData: log.Data{
				"log": "data",
			},
		}

		Convey("When logData(err) is called", func() {
			ld := logData(err)
			So(ld, ShouldResemble, log.Data{"log": "data"})
		})
	})

	Convey("Given an error chain with wrapped logData", t, func() {
		err1 := &testError{
			err: errors.New("original error"),
			logData: log.Data{
				"log": "data",
			},
		}

		err2 := &testError{
			err: fmt.Errorf("err1: %w", err1),
			logData: log.Data{
				"additional": "data",
			},
		}

		err3 := &testError{
			err: fmt.Errorf("err2: %w", err2),
			logData: log.Data{
				"final": "data",
			},
		}

		Convey("When unwrapLogData(err) is called", func() {
			logData := unwrapLogData(err3)
			expected := log.Data{
				"final":      "data",
				"additional": "data",
				"log":        "data",
			}

			So(logData, ShouldResemble, expected)
		})
	})

	Convey("Given an error chain with intermittent wrapped logData", t, func() {
		err1 := &testError{
			err: errors.New("original error"),
			logData: log.Data{
				"log": "data",
			},
		}

		err2 := &testError{
			err: fmt.Errorf("err1: %w", err1),
		}

		err3 := &testError{
			err: fmt.Errorf("err2: %w", err2),
			logData: log.Data{
				"final": "data",
			},
		}

		Convey("When unwrapLogData(err) is called", func() {
			logData := unwrapLogData(err3)
			expected := log.Data{
				"final": "data",
				"log":   "data",
			}

			So(logData, ShouldResemble, expected)
		})
	})

	Convey("Given an error chain with wrapped logData with duplicate key values", t, func() {
		err1 := &testError{
			err: errors.New("original error"),
			logData: log.Data{
				"log":        "data",
				"duplicate":  "duplicate_data1",
				"request_id": "ADB45F",
			},
		}

		err2 := &testError{
			err: fmt.Errorf("err1: %w", err1),
			logData: log.Data{
				"additional": "data",
				"duplicate":  "duplicate_data2",
				"request_id": "ADB45F",
			},
		}

		err3 := &testError{
			err: fmt.Errorf("err2: %w", err2),
			logData: log.Data{
				"final":      "data",
				"duplicate":  "duplicate_data3",
				"request_id": "ADB45F",
			},
		}

		Convey("When unwrapLogData(err) is called", func() {
			logData := unwrapLogData(err3)
			expected := log.Data{
				"final":      "data",
				"additional": "data",
				"log":        "data",
				"duplicate": []interface{}{
					"duplicate_data3",
					"duplicate_data2",
					"duplicate_data1",
				},
				"request_id": "ADB45F",
			}

			So(logData, ShouldResemble, expected)
		})
	})
}
