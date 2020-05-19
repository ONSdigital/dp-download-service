package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/ONSdigital/dp-download-service/downloads"
	"github.com/ONSdigital/go-ns/common"
	"github.com/ONSdigital/log.go/log"
	"github.com/gorilla/mux"
)

// mockgen is prefixing the imports within the mock file with the vendor directory 'github.com/ONSdigital/dp-download-service/vendor/'
//go:generate mockgen -destination mocks/mocks.go -package mocks github.com/ONSdigital/dp-download-service/handlers DatasetDownloads,S3Content
//go:generate sed -i "" -e s!\([[:space:]]\"\)github.com/ONSdigital/dp-download-service/vendor/!\1! mocks/mocks.go

const (
	notFoundMessage       = "resource not found"
	internalServerMessage = "internal server error"
)

// ClientError implements error interface with additional code method
type ClientError interface {
	error
	Code() int
}

// IdentityClient is an interface to represent methods called to action on the identity api
type IdentityClient interface {
	CheckRequest(*http.Request, string, string)
}

// VaultClient is an interface to represent methods called to action upon vault
type VaultClient interface {
	ReadKey(path, key string) (string, error)
}

// S3Client is an interface to represent methods called to retrieve from s3
type S3Client interface {
	GetWithPSK(key string, psk []byte) (io.ReadCloser, error)
}

type S3Content interface {
	StreamAndWrite(ctx context.Context, filename string, w io.Writer) error
}

type DatasetDownloads interface {
	GetFilterOutputDownloads(ctx context.Context, p downloads.Parameters) (downloads.Model, error)
	GetDatasetVersionDownloads(ctx context.Context, p downloads.Parameters) (downloads.Model, error)
}

// Info represents the configuration for a download handler
type Download struct {
	DatasetDownloads     DatasetDownloads
	S3Content            S3Content
	ServiceAuthToken     string
	DownloadServiceToken string
	SecretKey            string
	IsPublishing         bool
}

func setStatusCode(ctx context.Context, w http.ResponseWriter, err error, logData log.Data) {
	status := http.StatusInternalServerError
	message := internalServerMessage

	var cliErr ClientError
	if errors.As(err, &cliErr) {
		status = cliErr.Code()
	}

	logData["setting_response_status"] = status
	logData["error"] = err.Error()
	log.Event(ctx, "setting status code for an error", log.INFO, logData)
	if status == http.StatusNotFound {
		message = notFoundMessage
	}
	http.Error(w, message, status)
}

// Do handles the retrieval of a requested file, by first calling the datasetID to see if
// the version has a public link available and redirecting if so, otherwise it decrypts the private
// file on the fly (if it is published). Authenticated requests will always allow access to the private,
// whether or not the version is published.
func (d Download) Do(extension, serviceAuthToken, downloadServiceToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		params := getDownloadParameters(req, serviceAuthToken, downloadServiceToken)
		logData := downloadParametersToLogData(params)

		var downloads downloads.Model
		var err error

		if len(params.FilterOutputID) > 0 {
			downloads, err = d.DatasetDownloads.GetFilterOutputDownloads(ctx, params)
		} else {
			downloads, err = d.DatasetDownloads.GetDatasetVersionDownloads(ctx, params)
		}

		if err != nil {
			setStatusCode(ctx, w, err, logData)
			return
		}

		logData["published"] = downloads.IsPublished
		log.Event(req.Context(), "attempting to get download", log.INFO, logData)

		authorised, logData := d.authenticate(req, logData)
		logData["authorised"] = authorised

		if len(downloads.Available[extension].Public) > 0 && downloads.IsPublished {
			http.Redirect(w, req, downloads.Available[extension].Public, http.StatusMovedPermanently)
			return
		}

		if len(downloads.Available[extension].Private) > 0 {

			logData["private_link"] = downloads.Available[extension].Private
			log.Event(req.Context(), "using private link", log.INFO, logData)

			if downloads.IsPublished || authorised {

				privateFile := downloads.Available[extension].Private

				privateURL, err := url.Parse(privateFile)
				if err != nil {
					setStatusCode(ctx, w, err, logData)
					return
				}

				filename := privateURL.Path
				logData["filename"] = filename

				err = d.S3Content.StreamAndWrite(ctx, filename, w)
				if err != nil {
					setStatusCode(ctx, w, err, logData)
					return
				}

				log.Event(ctx, "download content successfully written to response", log.INFO, logData)
				return
			}
		}

		log.Event(ctx, "no public or private link found", log.ERROR, logData)
		http.Error(w, notFoundMessage, http.StatusNotFound)
	}
}

func getDownloadParameters(req *http.Request, serviceAuthToken, downloadServiceToken string) downloads.Parameters {
	vars := mux.Vars(req)

	return downloads.Parameters{
		UserAuthToken:        getUserAccessTokenFromContext(req.Context()),
		ServiceAuthToken:     serviceAuthToken,
		DownloadServiceToken: downloadServiceToken,
		CollectionID:         getCollectionIDFromContext(req.Context()),
		FilterOutputID:       vars["filterOutputID"],
		DatasetID:            vars["datasetID"],
		Edition:              vars["edition"],
		Version:              vars["version"],
	}
}

func downloadParametersToLogData(p downloads.Parameters) log.Data {
	logData := log.Data{}

	if len(p.CollectionID) > 0 {
		logData["collection_id"] = p.CollectionID
	}
	if len(p.FilterOutputID) > 0 {
		logData["filter_output_id"] = p.FilterOutputID
	}
	if len(p.DatasetID) > 0 {
		logData["dataset_id"] = p.DatasetID
	}
	if len(p.Edition) > 0 {
		logData["edition"] = p.Edition
	}
	if len(p.Version) > 0 {
		logData["version"] = p.Version
	}

	return logData
}

func (d Download) authenticate(r *http.Request, logData map[string]interface{}) (bool, map[string]interface{}) {
	var authorised bool

	if d.IsPublishing {
		authorised = common.IsCallerPresent(r.Context())
	}

	logData["authenticated"] = authorised
	return authorised, logData
}

func getUserAccessTokenFromContext(ctx context.Context) string {
	if ctx.Value(common.FlorenceIdentityKey) != nil {
		accessToken, ok := ctx.Value(common.FlorenceIdentityKey).(string)
		if !ok {
			log.Event(ctx, "access token error", log.ERROR, log.Error(errors.New("error casting access token context value to string")))
		}
		return accessToken
	}
	return ""
}

func getCollectionIDFromContext(ctx context.Context) string {
	if ctx.Value(common.CollectionIDHeaderKey) != nil {
		collectionID, ok := ctx.Value(common.CollectionIDHeaderKey).(string)
		if !ok {
			log.Event(ctx, "collection id error", log.ERROR, log.Error(errors.New("error casting collection ID context value to string")))
		}
		return collectionID
	}
	return ""
}
