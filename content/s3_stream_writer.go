package content

import (
	"context"
	"fmt"
	"io"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/log.go/v2/log"
)

//go:generate mockgen -destination=mocks/mocks.go -package=mocks github.com/ONSdigital/dp-download-service/content Writer,S3Client,S3ReadCloser

// Writer is an io.Writer alias to allow mockgen to create a mock impl for the tests
type Writer io.Writer

// S3ReadCloser is an io.ReadCloser alias to allow mockgen to create a mock impl for the tests
type S3ReadCloser io.ReadCloser

// S3Client is an interface to represent methods called to retrieve from s3
type S3Client interface {
	Get(ctx context.Context, key string) (io.ReadCloser, *int64, error)
	Checker(ctx context.Context, check *healthcheck.CheckState) error
}

// S3StreamWriter provides functionality for retrieving content from an S3 bucket. The content is streamed and and written to the provided io.Writer
type S3StreamWriter struct {
	S3Client S3Client
}

// NewStreamWriter create a new S3StreamWriter instance.
func NewStreamWriter(s3c S3Client) *S3StreamWriter {
	return &S3StreamWriter{
		S3Client: s3c,
	}
}

// StreamAndWrite stream the request file writing the content to the provided io.Writer.
func (s S3StreamWriter) StreamAndWrite(ctx context.Context, s3Path string, w io.Writer) (err error) {
	var s3ReadCloser io.ReadCloser
	s3ReadCloser, _, err = s.S3Client.Get(ctx, s3Path)
	if err != nil {
		return fmt.Errorf("failed to get stream object from S3 client: %w", err)
	}

	defer closeAndLogError(ctx, s3ReadCloser)

	_, err = io.Copy(w, s3ReadCloser)
	if err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}

func closeAndLogError(ctx context.Context, closer io.Closer) {
	if err := closer.Close(); err != nil {
		log.Error(ctx, "error closing io.Closer", err)
	}
}
