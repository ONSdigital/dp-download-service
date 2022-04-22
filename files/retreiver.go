package files

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	dprequest "github.com/ONSdigital/dp-net/request"
	"io"
	"net/http"

	"github.com/ONSdigital/dp-download-service/content"
)

const (
	VAULT_KEY = "key"
)

type HTTPClient interface {
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
}

var ErrFileNotRegistered = errors.New("file not registered")
var ErrBadJSONResponse = errors.New("could not decode JSON response from files api")
var ErrNotAuthorised = errors.New("the request was not authorised - check token and user's permissions")
var ErrInternalServerError = errors.New("internal server error")
var ErrUnknown = errors.New("an unknown error occurred")

type FileDownloader func(path string) (io.ReadCloser, error)
type MetadataFetcher func(ctx context.Context, path string) (Metadata, error)

type ContextKey string

func FetchMetadata(filesApiUrl string, httpClient HTTPClient) MetadataFetcher {
	return func(ctx context.Context, path string) (Metadata, error) {
		m := Metadata{}

		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/files/%s", filesApiUrl, path), nil)
		const authKey ContextKey = dprequest.AuthHeaderKey
		authHeaderValue := ctx.Value(authKey)
		if authHeaderValue != nil {
			req.Header.Add(dprequest.AuthHeaderKey, authHeaderValue.(string))
		}

		resp, _ := httpClient.Do(ctx, req)

		switch resp.StatusCode {
		case http.StatusOK:
			if json.NewDecoder(resp.Body).Decode(&m) != nil {
				return m, ErrBadJSONResponse
			}
			return m, nil
		case http.StatusNotFound:
			return m, ErrFileNotRegistered
		case http.StatusInternalServerError:
			return m, ErrInternalServerError
		case http.StatusForbidden:
			return m, ErrNotAuthorised
		default:
			return m, ErrUnknown
		}
	}
}

func DownloadFile(s3client content.S3Client, vc content.VaultClient, vaultPath string) FileDownloader {
	return func(filePath string) (io.ReadCloser, error) {
		pskStr, err := vc.ReadKey(fmt.Sprintf("%s/%s", vaultPath, filePath), VAULT_KEY)
		if err != nil {
			return nil, err
		}

		encryptionKey, err := hex.DecodeString(pskStr)
		if err != nil {
			return nil, err
		}

		file, _, err := s3client.GetWithPSK(filePath, encryptionKey)

		return file, err
	}
}
