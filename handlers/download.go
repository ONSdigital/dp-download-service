package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/ONSdigital/dp-download-service/downloads"
	dphandlers "github.com/ONSdigital/dp-net/v2/handlers"
	"github.com/ONSdigital/dp-net/v2/request"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/gorilla/mux"
)

//go:generate mockgen -destination mocks/mocks.go -package mocks github.com/ONSdigital/dp-download-service/handlers Downloader,S3Content

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

// S3Content is an interface to represent methods called to action on S3
type S3Content interface {
	StreamAndWrite(ctx context.Context, s3Path string, vaultPath string, w io.Writer) error
}

// Downloader is an interface to represent methods called to obtain the download metadata for any possible download type (dataset, image, etc)
type Downloader interface {
	Get(ctx context.Context, p downloads.Parameters, fileType downloads.FileType, variant string) (downloads.Model, error)
}

// Download represents the configuration for a download handler
type Download struct {
	Downloader           Downloader
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
	log.Info(ctx, "setting status code for an error", logData)
	if status == http.StatusNotFound {
		message = notFoundMessage
	}
	http.Error(w, message, status)
}

func (d Download) DoInstance(extension, serviceAuthToken, downloadServiceToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		params := GetDownloadParameters(req, serviceAuthToken, downloadServiceToken)
		d.do(w, req, downloads.TypeInstance, params, extension)
	}
}

// DoImage handles download image file requests.
func (d Download) DoImage(serviceAuthToken, downloadServiceToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		params := GetDownloadParameters(req, serviceAuthToken, downloadServiceToken)
		d.do(w, req, downloads.TypeImage, params, params.Variant)
	}
}

// DoDatasetVersion handles dataset version file download requests.
func (d Download) DoDatasetVersion(extension, serviceAuthToken, downloadServiceToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		params := GetDownloadParameters(req, serviceAuthToken, downloadServiceToken)
		d.do(w, req, downloads.TypeDatasetVersion, params, extension)
	}
}

// DoFilterOutput handles filter outpout download requests.
func (d Download) DoFilterOutput(extension, serviceAuthToken, downloadServiceToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		params := GetDownloadParameters(req, serviceAuthToken, downloadServiceToken)
		d.do(w, req, downloads.TypeFilterOutput, params, extension)
	}
}

// do handles download requests for any possible provided file type. If the object is published and a public download link is available then
// the request is redirected to the existing public link.
// If the object is published but a public link does not exist then the requested file is streamed from the content
// store and written to response body.
// Authenticated requests will always allow access to the private, whether or not the version is published.
func (d Download) do(w http.ResponseWriter, req *http.Request, fileType downloads.FileType, params downloads.Parameters, variant string) {
	ctx := req.Context()
	logData := downloadParametersToLogData(params)

	var err error

	fileDownloads, err := d.Downloader.Get(ctx, params, fileType, variant)
	if err != nil {
		setStatusCode(ctx, w, err, logData)
		return
	}

	logData["published"] = fileDownloads.IsPublished
	log.Info(req.Context(), "attempting to get download", logData)

	authorised, logData := d.authenticate(req, logData)
	logData["authorised"] = authorised

	if fileDownloads.IsPublicLinkAvailable() {
		http.Redirect(w, req, fileDownloads.Public, http.StatusMovedPermanently)
		return
	}

	if len(fileDownloads.PrivateS3Path) > 0 {
		s3Path := fileDownloads.PrivateS3Path
		vaultPath := fileDownloads.PrivateVaultPath
		filename := fileDownloads.PrivateFilename

		logData["private_s3_path"] = s3Path
		logData["private_vault_path"] = vaultPath
		logData["private_filename"] = filename
		log.Info(req.Context(), "using private link", logData)

		if fileDownloads.IsPublished || authorised {
			w.Header().Set("Content-Disposition", "attachment; filename="+filename)

			err = d.S3Content.StreamAndWrite(ctx, s3Path, vaultPath, w)
			if err != nil {
				setStatusCode(ctx, w, err, logData)
				return
			}

			log.Info(ctx, "download content successfully written to response", logData)
			return
		}
	}

	log.Error(ctx, "no public or private link found", errors.New("no public or private link found"), logData)
	http.Error(w, notFoundMessage, http.StatusNotFound)
}

// GetDownloadParameters extracts the query parameters and context values for the provided request,
// then returns a struct with all the available parameters, including the explicitly provided service and downloadService tokens
func GetDownloadParameters(req *http.Request, serviceAuthToken, downloadServiceToken string) downloads.Parameters {
	vars := mux.Vars(req)

	return downloads.Parameters{
		UserAuthToken:        getUserAccessTokenFromContext(req.Context()),
		ServiceAuthToken:     serviceAuthToken,
		DownloadServiceToken: downloadServiceToken,
		CollectionID:         getCollectionIDFromContext(req.Context()),
		FilterOutputID:       vars["filterOutputID"],
		InstanceID:           vars["instanceID"],
		DatasetID:            vars["datasetID"],
		Edition:              vars["edition"],
		Version:              vars["version"],
		ImageID:              vars["imageID"],
		Variant:              vars["variant"],
		Filename:             vars["filename"],
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
	if len(p.Filename) > 0 {
		logData["filename"] = p.Filename
	}

	return logData
}

func (d Download) authenticate(r *http.Request, logData map[string]interface{}) (bool, map[string]interface{}) {
	var authorised bool

	if d.IsPublishing {
		authorised = request.IsCallerPresent(r.Context())
	}

	logData["authenticated"] = authorised
	return authorised, logData
}

func getUserAccessTokenFromContext(ctx context.Context) string {
	if ctx.Value(dphandlers.UserAccess.Context()) != nil {
		accessToken, ok := ctx.Value(dphandlers.UserAccess.Context()).(string)
		if !ok {
			log.Error(ctx, "access token error", errors.New("error casting access token context value to string"))
		}
		return accessToken
	}
	return ""
}

func getCollectionIDFromContext(ctx context.Context) string {
	if ctx.Value(dphandlers.CollectionID.Context()) != nil {
		collectionID, ok := ctx.Value(dphandlers.CollectionID.Context()).(string)
		if !ok {
			log.Error(ctx, "collection id error", errors.New("error casting collection ID context value to string"))
		}
		return collectionID
	}
	return ""
}
