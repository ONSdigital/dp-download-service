package content

import (
	"context"
	"encoding/hex"
	"errors"
	"io"

	"github.com/ONSdigital/log.go/log"
)

var (
	VaultFilenameEmptyErr = errors.New("vault filename required but was empty")
	vaultKey              = "key"
)

//go:generate mockgen -destination mocks/mocks.go -package mocks github.com/ONSdigital/dp-download-service/content VaultClient,Writer,S3Client,S3ReadCloser

// Writer is an io.Writer alias to allow mockgen to create a mock impl for the tests
type Writer io.Writer

// S3ReadCloser is an io.ReadCloser alias to allow mockgen to create a mock impl for the tests
type S3ReadCloser io.ReadCloser

// VaultClient is an interface to represent methods called to action upon vault
type VaultClient interface {
	ReadKey(path, key string) (string, error)
}

// S3Client is an interface to represent methods called to retrieve from s3
type S3Client interface {
	Get(key string) (io.ReadCloser, *int64, error)
	GetWithPSK(key string, psk []byte) (io.ReadCloser, *int64, error)
}

//S3StreamWriter provides functionality for retrieving content from an S3 bucket. The content is streamed/decrypted and and written to the provided io.Writer
type S3StreamWriter struct {
	VaultCli           VaultClient
	VaultPath          string
	S3Client           S3Client
	EncryptionDisabled bool
}

//NewStreamWriter create a new S3StreamWriter instance.
func NewStreamWriter(s3c S3Client, vc VaultClient, vp string, encDisabled bool) *S3StreamWriter {
	return &S3StreamWriter{
		S3Client:           s3c,
		VaultCli:           vc,
		VaultPath:          vp,
		EncryptionDisabled: encDisabled,
	}
}

//StreamAndWrite decrypt and stream the request file writing the content to the provided io.Writer.
func (s S3StreamWriter) StreamAndWrite(ctx context.Context, s3Path string, vaultPath string, w io.Writer) (err error) {
	var s3ReadCloser io.ReadCloser
	if s.EncryptionDisabled {
		s3ReadCloser, _, err = s.S3Client.Get(s3Path)
		if err != nil {
			return err
		}
	} else {
		psk, err := s.getVaultKeyForFile(vaultPath)
		if err != nil {
			return err
		}

		s3ReadCloser, _, err = s.S3Client.GetWithPSK(s3Path, psk)
		if err != nil {
			return err
		}
	}

	defer close(ctx, s3ReadCloser)

	_, err = io.Copy(w, s3ReadCloser)
	if err != nil {
		return err
	}

	return nil
}

func (s *S3StreamWriter) getVaultKeyForFile(secretPath string) ([]byte, error) {
	if len(secretPath) == 0 {
		return nil, VaultFilenameEmptyErr
	}

	vp := s.VaultPath + "/" + secretPath
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
