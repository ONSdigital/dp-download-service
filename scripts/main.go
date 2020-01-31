package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"io/ioutil"
	"os"

	"github.com/ONSdigital/go-ns/vault"
	"github.com/ONSdigital/log.go/log"
	"github.com/ONSdigital/s3crypto"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var filename = "2470609-cpicoicoptestcsv"
var bucket = "dp-frontend-florence-file-uploads"

func main() {
	vaultAddress := os.Getenv("VAULT_ADDR")
	token := os.Getenv("VAULT_TOKEN")

	client, err := vault.CreateVaultClient(token, vaultAddress, 3)

	ctx := context.Background()
	logData := log.Data{"address": vaultAddress}
	log.Event(ctx, "Created vault client", logData)

	if err != nil {
		log.Event(ctx, "failed to connect to vault", log.Error(err), logData)
		return
	}

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

	input := &s3.PutObjectInput{
		Body:   rs,
		Key:    &filename,
		Bucket: &bucket,
	}

	region := "eu-west-1"

	sess, err := session.NewSession(&aws.Config{Region: &region})
	if err != nil {
		log.Event(ctx, "error creating new session", log.Error(err))
		return
	}
	s3cli := s3crypto.New(sess, &s3crypto.Config{HasUserDefinedPSK: true})

	_, err = s3cli.PutObjectWithPSK(input, psk)
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
