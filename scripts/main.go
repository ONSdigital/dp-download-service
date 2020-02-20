package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"io/ioutil"
	"os"

	s3client "github.com/ONSdigital/dp-s3"
	vault "github.com/ONSdigital/dp-vault"
	"github.com/ONSdigital/log.go/log"
)

var filename = "2470609-cpicoicoptestcsv"
var bucket = "dp-frontend-florence-file-uploads"
var region = "eu-west-1"

func main() {
	vaultAddress := os.Getenv("VAULT_ADDR")
	token := os.Getenv("VAULT_TOKEN")

	ctx := context.Background()
	logData := log.Data{"address": vaultAddress}

	client, err := vault.CreateClient(token, vaultAddress, 3)
	if err != nil {
		log.Event(ctx, "failed to connect to vault", log.Error(err), logData)
		return
	}
	log.Event(ctx, "Created vault client", logData)

	psk := createPSK()
	pskStr := hex.EncodeToString(psk)

	if err := client.WriteKey("secret/shared/psk", filename, pskStr); err != nil {
		log.Event(ctx, "error writting key", log.Error(err))
		return
	}

	b, err := ioutil.ReadFile("cpicoicoptest.csv")
	if err != nil {
		log.Event(ctx, "failed to connect to vault", log.Error(err), logData)
		return
	}
	rs := bytes.NewReader(b)

	s3cli, err := s3client.NewClient(region, bucket, true)
	if err != nil {
		log.Event(ctx, "error creating new S3 client", log.Error(err))
		return
	}

	err = s3cli.PutWithPSK(&filename, rs, psk)
	if err != nil {
		log.Event(ctx, "error putting object with PSK", log.Error(err))
		return
	}

	log.Event(ctx, "file encrypted and uploaded to s3", log.Data{"file": filename})

}

func createPSK() []byte {
	key := make([]byte, 16)
	rand.Read(key)

	return key
}
