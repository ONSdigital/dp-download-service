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

type S3Content interface {
	StreamAndWrite(ctx context.Context, filename string, w io.Writer) error
}

type DatasetDownloads interface {
	Get(ctx context.Context, p downloads.Parameters) (downloads.Model, error)
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

// DoImage handle download image file requests.
func (d Download) DoImage(serviceAuthToken, downloadServiceToken string) http.HandlerFunc {
	// router.Path("/images/{id}/{variant}/{name}.{ext}").HandlerFunc(d.DoImage(cfg.ServiceAuthToken, cfg.DownloadServiceToken))
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		params := getDownloadParameters(req, serviceAuthToken, downloadServiceToken)
		logData := downloadParametersToLogData(params)
		log.Event(ctx, "download image request", log.INFO, logData)
	}
}

// DoDataset handle download dataset file requests. If the dataset is published and a public download link is available then
// the request is redirected to the existing public link.
// If the dataset is published but a public link does not exist then the requested file is streamed from the content
// store and written to response body.
// Authenticated requests will always allow access to the private, whether or not the version is published.
func (d Download) DoDataset(extension, serviceAuthToken, downloadServiceToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		params := getDownloadParameters(req, serviceAuthToken, downloadServiceToken)
		logData := downloadParametersToLogData(params)

		var err error

		datasetDownloads, err := d.DatasetDownloads.Get(ctx, params)
		if err != nil {
			setStatusCode(ctx, w, err, logData)
			return
		}

		logData["published"] = datasetDownloads.IsPublished
		log.Event(req.Context(), "attempting to get download", log.INFO, logData)

		authorised, logData := d.authenticate(req, logData)
		logData["authorised"] = authorised

		if datasetDownloads.IsPublicLinkAvailable(extension) {
			http.Redirect(w, req, datasetDownloads.Available[extension].Public, http.StatusMovedPermanently)
			return
		}

		if len(datasetDownloads.Available[extension].Private) > 0 {

			logData["private_link"] = datasetDownloads.Available[extension].Private
			log.Event(req.Context(), "using private link", log.INFO, logData)

			if datasetDownloads.IsPublished || authorised {

				privateFile := datasetDownloads.Available[extension].Private

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
		ImageID:              vars["imageID"],
		Variant:              vars["variant"],
		Name:                 vars["name"],
		Ext:                  vars["ext"],
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
	if len(p.ImageID) > 0 {
		logData["imageID"] = p.ImageID
	}
	if len(p.Variant) > 0 {
		logData["variant"] = p.Variant
	}
	if len(p.Name) > 0 {
		logData["name"] = p.Name
	}
	if len(p.Ext) > 0 {
		logData["ext"] = p.Ext
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
