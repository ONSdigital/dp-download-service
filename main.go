package main

import (
	"encoding/hex"
	"io/ioutil"
	"net/http"

	"github.com/ONSdigital/go-ns/server"
	"github.com/ONSdigital/go-ns/vault"

	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/go-ns/clients/dataset"
	"github.com/ONSdigital/go-ns/log"
	"github.com/ONSdigital/s3crypto"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
)

const internalToken = "internal-token"

func main() {
	cfg, err := config.Get()
	if err != nil {
		log.Error(err, nil)
		return
	}

	client, err := vault.CreateVaultClient(cfg.VaultToken, cfg.VaultAddress, 3)
	if err != nil {
		log.Error(err, nil)
		return
	}
	log.Info("vault client created", log.Data{"vault_addr": cfg.VaultAddress})

	log.Info("service config", log.Data{
		"bind_address":        cfg.BindAddr,
		"dataset_api_url":     cfg.DatasetAPIURL,
		"encryption_disabled": cfg.EncryptionDisabled,
	})

	router := mux.NewRouter()

	dc := dataset.New(cfg.DatasetAPIURL)
	dc.SetInternalToken(cfg.DatasetAuthToken)

	region := "eu-west-1"

	sess, err := session.NewSession(&aws.Config{Region: &region})
	if err != nil {
		log.Error(err, nil)
		return
	}

	svc := &Service{dc, s3crypto.New(
		sess,
		&s3crypto.Config{HasUserDefinedPSK: true}),
		"dp-frontend-florence-file-uploads",
		cfg.SecretKey,
		client,
	}

	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.csv").HandlerFunc(svc.getFile("csv"))
	router.Path("/downloads/datasets/{datasetID}/editions/{edition}/versions/{version}.xls").HandlerFunc(svc.getFile("xls"))

	s := server.New(cfg.BindAddr, router)

	s.ListenAndServe()
}

type Service struct {
	DatasetClient     *dataset.Client
	S3Client          *s3crypto.CryptoClient
	PrivateBucketName string
	SecretKey         string
	VaultClient       *vault.VaultClient
}

func (svc *Service) getFile(extension string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		id := vars["datasetID"]
		edition := vars["edition"]
		version := vars["version"]

		log.InfoR(req, "attempting to get download", log.Data{
			"dataset_id": id,
			"edition":    edition,
			"version":    version,
		})

		v, err := svc.DatasetClient.GetVersion(id, edition, version)
		if err != nil {
			log.ErrorR(req, err, nil)
			return
		}

		if len(v.Downloads[extension].Public) > 0 && v.State == "published" {
			http.Redirect(w, req, v.Downloads["csv"].Public, 302)
			return
		}

		if len(v.Downloads[extension].Private) > 0 {
			if v.State == "published" || req.Header.Get(internalToken) == svc.SecretKey {
				privateFile := v.Downloads[extension].Private

				input := &s3.GetObjectInput{
					Bucket: &svc.PrivateBucketName,
					Key:    &privateFile,
				}

				pskStr, err := svc.VaultClient.ReadKey("secret/shared/psk", privateFile)
				if err != nil {
					log.ErrorR(req, err, nil)
					return
				}
				psk, err := hex.DecodeString(pskStr)
				if err != nil {
					log.ErrorR(req, err, nil)
					return
				}

				output, err := svc.S3Client.GetObjectWithPSK(input, psk)
				if err != nil {
					log.ErrorR(req, err, nil)
					return
				}

				b, err := ioutil.ReadAll(output.Body)
				if err != nil {
					log.ErrorR(req, err, nil)
					return
				}

				defer output.Body.Close()

				w.Write(b)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}

	}
}
