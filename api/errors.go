package api

import (
	"encoding/json"
	"github.com/ONSdigital/dp-download-service/files"
	"net/http"
)

//nolint:golint,unused
type jsonError struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

//nolint:golint,unused
type jsonErrors struct {
	Error []jsonError `json:"errors"`
}

//nolint:golint,unused
func handleError(w http.ResponseWriter, err error) {
	switch err {
	case files.ErrFileNotRegistered:
		writeError(w, buildErrors(err, "FileNotRegistered"), http.StatusNotFound)
	default:
		writeError(w, buildErrors(err, "InternalError"), http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, errs jsonErrors, httpCode int) {
	encoder := json.NewEncoder(w)
	w.WriteHeader(httpCode)
	encoder.Encode(&errs) // nolint
}

func buildErrors(err error, code string) jsonErrors {
	return jsonErrors{Error: []jsonError{{Description: err.Error(), Code: code}}}
}
