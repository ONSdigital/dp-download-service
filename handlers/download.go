package handlers

import (
	"context"
	"encoding/hex"
	"io"
	"net/http"
	"path/filepath"

	"github.com/ONSdigital/go-ns/clients/dataset"
	"github.com/ONSdigital/go-ns/clients/filter"
	"github.com/ONSdigital/go-ns/log"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
	"github.com/ONSdigital/go-ns/common"
)

// mockgen is prefixing the imports within the mock file with the vendor directory 'github.com/ONSdigital/dp-download-service/vendor/'
// I just manually removed it, after spending a bit of time trying to find a clean solution
//go:generate mockgen -destination mocks/mocks.go -package mocks github.com/ONSdigital/dp-download-service/handlers DatasetClient,VaultClient,S3Client,FilterClient

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
	GetVersion(ctx context.Context, id, edition, version string) (m dataset.Version, err error)
}

// FilterClient is an interface to represent methods called to action on the filter api
type FilterClient interface {
	GetOutput(ctx context.Context, filterOutputID string) (m filter.Model, err error)
}

// VaultClient is an interface to represent methods called to action upon vault
type VaultClient interface {
	ReadKey(path, key string) (string, error)
}

// S3Client is an interface to represent methods called to retrieve from s3
type S3Client interface {
	GetObjectWithPSK(*s3.GetObjectInput, []byte) (*s3.GetObjectOutput, error)
}

type download struct {
	URL     string `json:"url"`
	Size    string `json:"size"`
	Public  string `json:"public"`
	Private string `json:"private"`
}

// Download represents the configuration for a download handler
type Download struct {
	DatasetClient             DatasetClient
	VaultClient               VaultClient
	FilterClient              FilterClient
	S3Client                  S3Client
	ServiceToken              string
	DatasetAuthToken          string
	XDownloadServiceAuthToken string
	SecretKey                 string
	BucketName                string
	VaultPath                 string
	IsPublishing              bool
}

func setStatusCode(req *http.Request, w http.ResponseWriter, err error, logData log.Data) {
	status := http.StatusInternalServerError
	message := internalServerMessage
	if err, ok := err.(ClientError); ok {
		status = err.Code()
	}
	logData["setting_response_status"] = status
	log.ErrorR(req, err, logData)
	if status == http.StatusNotFound {
		message = notFoundMessage
	}
	http.Error(w, message, status)
}

// Do handles the retrieval of a requested file, by first calling the datasetID to see if
// the version has a public link available and redirecting if so, otherwise it decrypts the private
// file on the fly (if it is published). Authenticated requests will always allow access to the private,
// whether or not the version is published.
func (d Download) Do(extension string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		datasetID := vars["datasetID"]
		edition := vars["edition"]
		version := vars["version"]
		filterOutputID := vars["filterOutputID"]

		logData := log.Data{}
		published := false
		downloads := make(map[string]download)

		if len(filterOutputID) > 0 {
			logData = log.Data{
				"filter_output_id": filterOutputID,
				"type":             extension,
			}

			fo, err := d.FilterClient.GetOutput(req.Context(), filterOutputID)
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

			v, err := d.DatasetClient.GetVersion(req.Context(), datasetID, edition, version)
			if err != nil {
				setStatusCode(req, w, err, logData)
				return
			}

			published = v.State == "published"

			for k, v := range v.Downloads {
				downloads[k] = download(v)
			}
		}

		log.InfoR(req, "attempting to get download", logData)

		authorised, logData := d.authenticate(req, logData)

		if len(downloads[extension].Public) > 0 && published {
			http.Redirect(w, req, downloads[extension].Public, http.StatusMovedPermanently)
			return
		}

		if len(downloads[extension].Private) > 0 {

			if published || authorised {

				privateFile := downloads[extension].Private
				filename := filepath.Base(privateFile)

				input := &s3.GetObjectInput{
					Bucket: &d.BucketName,
					Key:    &filename,
				}

				vaultPath := d.VaultPath + "/" + filename
				vaultKey := "key"
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

				obj, err := d.S3Client.GetObjectWithPSK(input, psk)
				if err != nil {
					setStatusCode(req, w, err, logData)
					return
				}

				defer func() {
					if err := obj.Body.Close(); err != nil {
						log.ErrorR(req, err, nil)
					}
				}()

				if _, err := io.Copy(w, obj.Body); err != nil {
					setStatusCode(req, w, err, logData)
				}

				return
			}
		}

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
