package handlers

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/ONSdigital/dp-api-clients-go/dataset"
	"github.com/ONSdigital/dp-api-clients-go/filter"
	"github.com/ONSdigital/go-ns/common"
	"github.com/ONSdigital/log.go/log"
	"github.com/gorilla/mux"
)

// mockgen is prefixing the imports within the mock file with the vendor directory 'github.com/ONSdigital/dp-download-service/vendor/'
//go:generate mockgen -destination mocks/mocks.go -package mocks github.com/ONSdigital/dp-download-service/handlers DatasetClient,VaultClient,S3Client,FilterClient
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

// DatasetClient is an interface to represent methods called to action on the dataset api
type DatasetClient interface {
	GetVersion(ctx context.Context, userAuthToken, serviceAuthToken, downloadServiceToken, collectionID, datasetID, edition, version string) (m dataset.Version, err error)
}

// FilterClient is an interface to represent methods called to action on the filter api
type FilterClient interface {
	GetOutput(ctx context.Context, userAuthToken, serviceAuthToken, downloadServiceToken, collectionID, filterOutputID string) (m filter.Model, err error)
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

type download struct {
	URL     string `json:"href"`
	Size    string `json:"size"`
	Public  string `json:"public,omitempty"`
	Private string `json:"private,omitempty"`
	Skipped bool   `json:"skipped,omitempty"`
}

// Download represents the configuration for a download handler
type Download struct {
	DatasetClient        DatasetClient
	VaultClient          VaultClient
	FilterClient         FilterClient
	S3Client             S3Client
	ServiceAuthToken     string
	DownloadServiceToken string
	SecretKey            string
	VaultPath            string
	IsPublishing         bool
}

func setStatusCode(req *http.Request, w http.ResponseWriter, err error, logData log.Data) {
	status := http.StatusInternalServerError
	message := internalServerMessage
	if err, ok := err.(ClientError); ok {
		status = err.Code()
	}
	logData["setting_response_status"] = status
	logData["error"] = err.Error()
	log.Event(req.Context(), "setting status code for an error", log.INFO, logData)
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
		vars := mux.Vars(req)
		datasetID := vars["datasetID"]
		edition := vars["edition"]
		version := vars["version"]
		filterOutputID := vars["filterOutputID"]

		logData := log.Data{}
		published := false
		downloads := make(map[string]download)
		userAuthToken := getUserAccessTokenFromContext(req.Context())
		collectionID := getCollectionIDFromContext(req.Context())

		if len(filterOutputID) > 0 {
			logData = log.Data{
				"filter_output_id": filterOutputID,
				"type":             extension,
			}

			fo, err := d.FilterClient.GetOutput(req.Context(), userAuthToken, serviceAuthToken, downloadServiceToken, collectionID, filterOutputID)
			if err != nil {
				setStatusCode(req, w, err, logData)
				return
			}

			published = fo.IsPublished

			for k, v := range fo.Downloads {
				downloads[k] = download(v)
			}

		} else {
			logData = log.Data{
				"dataset_id": datasetID,
				"edition":    edition,
				"version":    version,
				"type":       extension,
			}

			v, err := d.DatasetClient.GetVersion(req.Context(), userAuthToken, serviceAuthToken, downloadServiceToken, collectionID, datasetID, edition, version)
			if err != nil {
				setStatusCode(req, w, err, logData)
				return
			}

			published = v.State == "published"

			for k, v := range v.Downloads {
				datasetDownloadWithSkipped := download{
					URL:     v.URL,
					Size:    v.Size,
					Public:  v.Public,
					Private: v.Private,
					Skipped: false,
				}
				downloads[k] = datasetDownloadWithSkipped
			}
		}

		logData["published"] = published
		log.Event(req.Context(), "attempting to get download", log.INFO, logData)

		authorised, logData := d.authenticate(req, logData)
		logData["authorised"] = authorised

		if len(downloads[extension].Public) > 0 && published {
			http.Redirect(w, req, downloads[extension].Public, http.StatusMovedPermanently)
			return
		}

		if len(downloads[extension].Private) > 0 {

			logData["private_link"] = downloads[extension].Private
			log.Event(req.Context(), "using private link", log.INFO, logData)

			if published || authorised {

				privateFile := downloads[extension].Private

				privateURL, err := url.Parse(privateFile)
				if err != nil {
					setStatusCode(req, w, err, logData)
					return
				}

				filename := privateURL.Path
				logData["filename"] = filename

				vaultPath := d.VaultPath + "/" + filepath.Base(filename)
				vaultKey := "key"
				logData["vaultPath"] = vaultPath

				log.Event(req.Context(), "getting download key from vault", log.INFO, logData)
				pskStr, err := d.VaultClient.ReadKey(vaultPath, vaultKey)
				if err != nil {
					setStatusCode(req, w, err, logData)
					return
				}
				psk, err := hex.DecodeString(pskStr)
				if err != nil {
					setStatusCode(req, w, err, logData)
					return
				}

				log.Event(req.Context(), "getting file from s3", log.INFO, logData)
				s3Reader, err := d.S3Client.GetWithPSK(filename, psk)
				if err != nil {
					setStatusCode(req, w, err, logData)
					return
				}

				defer func() {
					if err := s3Reader.Close(); err != nil {
						log.Event(req.Context(), "error closing body", log.ERROR, log.Error(err))
					}
				}()

				if _, err := io.Copy(w, s3Reader); err != nil {
					setStatusCode(req, w, err, logData)
				}

				return
			}
		}

		log.Event(req.Context(), "no public or private link found", log.ERROR, logData)
		http.Error(w, notFoundMessage, http.StatusNotFound)
	}
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
