package handlers

// "dataset" handlers handle the new /downloads/<uuid> endpoints.
// The "downloads" name is already taken by the /dataset/edition/version
// paths.

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/ONSdigital/dp-download-service/model"
	"github.com/ONSdigital/log.go/v2/log"
)

// Generate mocks of dependencies
//
//go:generate moq -rm -pkg handlers_test -out moq_model_test.go . Model

// Model describes what we expect our underlying model to implement.
//
type Model interface {
	Create(ctx context.Context, payload *model.DatasetDocument) (string, error)
}

// Dataset implements dataset POST and PUT handlers.
//
type Dataset struct {
	model Model
}

// These messages are returned to client in error response bodies.
//
const (
	MsgCannotReadBody       = "cannot read request body"
	MsgCannotParseBody      = "cannot parse request body"
	MsgCannotCreateDocument = "cannot create document"
)

// NewDataset returns a new dataset handler which uses model for business logic.
//
func NewDataset(model Model) *Dataset {
	return &Dataset{
		model: model,
	}
}

// DoPostDatset is an http handler for POSTing dataset documents
//
func (ds *Dataset) DoPostDataset() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := context.Background()
		logData := log.Data{}

		// read request body
		//
		body, err := io.ReadAll(req.Body)
		if err != nil {
			logData["setting_response_status"] = http.StatusInternalServerError
			log.Error(ctx, MsgCannotReadBody, err, logData)
			errorResponse(ctx, w, http.StatusInternalServerError, MsgCannotReadBody)
			return
		}

		// parse request body (validation performed by model)
		//
		var payload model.DatasetDocument
		err = json.Unmarshal(body, &payload)
		if err != nil {
			logData["body"] = string(body)
			logData["setting_response_status"] = http.StatusBadRequest
			log.Error(ctx, MsgCannotParseBody, err, logData)
			errorResponse(ctx, w, http.StatusBadRequest, MsgCannotParseBody)
			return
		}

		// call model
		//
		uuid, err := ds.model.Create(ctx, &payload)
		if err != nil {
			status := http.StatusInternalServerError
			var cliErr ClientError
			if errors.As(err, &cliErr) {
				status = cliErr.Code()
			}
			logData["setting_response_status"] = status
			log.Error(ctx, MsgCannotCreateDocument, err, logData)
			errorResponse(ctx, w, status, MsgCannotCreateDocument)
			return
		}

		// format http response
		//
		createdResponse(ctx, w, uuid)
	}
}

// errorResponse emits a json document for errors
//
func errorResponse(ctx context.Context, w http.ResponseWriter, status int, msg string) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)

	response := struct {
		Error string `json:"error"`
	}{
		Error: msg,
	}
	buf, err := json.Marshal(&response)
	if err != nil {
		log.Error(ctx, "cannot marshal error response", err)
	}
	w.Write(buf)
}

// createResponse emits a json document for successful creation
//
func createdResponse(ctx context.Context, w http.ResponseWriter, uuid string) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := struct {
		ID string `json:"id"`
	}{
		ID: uuid,
	}
	buf, err := json.Marshal(&response)
	if err != nil {
		log.Error(ctx, "cannot marshal created response", err)
	}
	w.Write(buf)
}
