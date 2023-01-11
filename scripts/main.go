package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"

	s3client "github.com/ONSdigital/dp-s3"
	vault "github.com/ONSdigital/dp-vault"
	"github.com/ONSdigital/log.go/v2/log"
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
		log.Error(ctx, "failed to connect to vault", err, logData)
		return
	}
	log.Info(ctx, "created vault client", logData)
	psk := createPSK()
	pskStr := hex.EncodeToString(psk)

	if err := client.WriteKey("secret/shared/psk", filename, pskStr); err != nil {
		log.Error(ctx, "error writting key", err)
		return
	}

	b, err := os.ReadFile("cpicoicoptest.csv")
	if err != nil {
		log.Error(ctx, "failed to connect to vault", err, logData)
		return
	}
	rs := bytes.NewReader(b)

	s3cli, err := s3client.NewClient(region, bucket)
	if err != nil {
		log.Error(ctx, "error creating new s3 client", err)
		return
	}

	err = s3cli.PutWithPSK(&filename, rs, psk)
	if err != nil {
		log.Error(ctx, "error putting object with psk", err)
		return
	}

	log.Info(ctx, "file encrypted and uploaded to s3", log.Data{"file": filename})

}

func createPSK() []byte {
	key := make([]byte, 16)
	rand.Read(key) // nolint

	return key
}
