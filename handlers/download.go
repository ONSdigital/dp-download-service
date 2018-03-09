package handlers

import (
	"encoding/hex"
	"io"
	"net/http"
	"path/filepath"

	"github.com/ONSdigital/go-ns/clients/dataset"
	"github.com/ONSdigital/go-ns/log"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
)

const (
	internalToken         = "internal-token"
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
	GetVersion(id, edition, version string, cfg ...dataset.Config) (m dataset.Version, err error)
}

// VaultClient is an interface to represent methods called to action upon vault
type VaultClient interface {
	ReadKey(path, key string) (string, error)
}

// S3Client is an interface to represent methods called to retrieve from s3
type S3Client interface {
	GetObjectWithPSK(*s3.GetObjectInput, []byte) (*s3.GetObjectOutput, error)
}

// Download represents the configuration for a download handler
type Download struct {
	DatasetClient    DatasetClient
	VaultClient      VaultClient
	S3Client         S3Client
	DatasetAuthToken string
	SecretKey        string
	BucketName       string
	VaultPath        string
}

func setStatusCode(req *http.Request, w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := internalServerMessage
	if err, ok := err.(ClientError); ok {
		status = err.Code()
	}
	log.ErrorR(req, err, log.Data{"setting-response-status": status})
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

		log.InfoR(req, "attempting to get download", log.Data{
			"dataset_id": datasetID,
			"edition":    edition,
			"version":    version,
			"type":       extension,
		})

		reqConfig := dataset.Config{
			InternalToken: d.DatasetAuthToken,
			Ctx:           req.Context(),
		}

		v, err := d.DatasetClient.GetVersion(datasetID, edition, version, reqConfig)
		if err != nil {
			setStatusCode(req, w, err)
			return
		}

		if len(v.Downloads[extension].Public) > 0 && v.State == "published" {
			http.Redirect(w, req, v.Downloads[extension].Public, http.StatusMovedPermanently)
			return
		}

		if len(v.Downloads[extension].Private) > 0 {
			if v.State == "published" || req.Header.Get(internalToken) == d.SecretKey {

				privateFile := v.Downloads[extension].Private
				filename := filepath.Base(privateFile)

				input := &s3.GetObjectInput{
					Bucket: &d.BucketName,
					Key:    &filename,
				}

				pskStr, err := d.VaultClient.ReadKey(d.VaultPath, filename)
				if err != nil {
					setStatusCode(req, w, err)
					return
				}
				psk, err := hex.DecodeString(pskStr)
				if err != nil {
					setStatusCode(req, w, err)
					return
				}

				obj, err := d.S3Client.GetObjectWithPSK(input, psk)
				if err != nil {
					setStatusCode(req, w, err)
					return
				}

				defer func() {
					if err := obj.Body.Close(); err != nil {
						log.ErrorR(req, err, nil)
					}
				}()

				if _, err := io.Copy(w, obj.Body); err != nil {
					setStatusCode(req, w, err)
				}

				return
			}
		}

		http.Error(w, notFoundMessage, http.StatusNotFound)
	}
}
