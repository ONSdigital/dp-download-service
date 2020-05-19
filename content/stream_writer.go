package content

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"path/filepath"

	"github.com/ONSdigital/log.go/log"
)

var (
	VaultFilenameEmptyErr = errors.New("vault filename required but was empty")
	vaultKey              = "key"
)

//go:generate mockgen -destination mocks/mocks.go -package mocks github.com/ONSdigital/dp-download-service/content VaultClient,Writer,S3Client,S3ReadCloser

// aliased to allow mockgen to create a mock impl for the tests
type Writer io.Writer

// aliased to allow mockgen to create a mock impl for the tests
type S3ReadCloser io.ReadCloser

// VaultClient is an interface to represent methods called to action upon vault
type VaultClient interface {
	ReadKey(path, key string) (string, error)
}

// S3Client is an interface to represent methods called to retrieve from s3
type S3Client interface {
	GetWithPSK(key string, psk []byte) (io.ReadCloser, error)
}

//StreamWriter provides functionality for retrieving content from an S3 bucket. The content is streamed/decrypted and and written to the provided io.Writer
type StreamWriter struct {
	VaultCli  VaultClient
	VaultPath string
	S3Client  S3Client
}

//New create a new StreamWriter instance.
func New(vc VaultClient, vp string) *StreamWriter {
	return &StreamWriter{VaultCli: vc, VaultPath: vp}
}

//WriteContent write the content of specified file to the provided io.Writer.
func (s StreamWriter) WriteContent(ctx context.Context, filename string, w io.Writer) error {
	psk, err := s.getVaultKeyForFile(filename)
	if err != nil {
		return err
	}

	s3ReadCloser, err := s.S3Client.GetWithPSK(filename, psk)
	if err != nil {
		return err
	}

	defer close(ctx, s3ReadCloser)

	_, err = io.Copy(w, s3ReadCloser)
	if err != nil {
		return err
	}

	return nil
}

func (s *StreamWriter) getVaultKeyForFile(filename string) ([]byte, error) {
	if len(filename) == 0 {
		return nil, VaultFilenameEmptyErr
	}

	vp := s.VaultPath + "/" + filepath.Base(filename)
	pskStr, err := s.VaultCli.ReadKey(vp, vaultKey)
	if err != nil {
		return nil, err
	}

	psk, err := hex.DecodeString(pskStr)
	if err != nil {
		return nil, err
	}

	return psk, nil
}

func close(ctx context.Context, closer io.Closer) {
	if err := closer.Close(); err != nil {
		log.Event(ctx, "error closing io.Closer", log.ERROR, log.Error(err))
	}
}
