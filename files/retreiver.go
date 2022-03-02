package files

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ONSdigital/dp-download-service/content"
	"io"
	"net/http"
)

const (
	VAULT_KEY = "key"
)

type HTTPClient interface {
	Get(url string) (resp *http.Response, err error)
}

var ErrFileNotRegistered = errors.New("file not registered")
var ErrBadJSONResponse = errors.New("could not decode JSON response from files api")

type FileDownloader func(path string) (io.ReadCloser, error)
type MetadataFetcher func(path string) (Metadata, error)

func FetchMetadata(filesApiUrl string, httpClient HTTPClient) MetadataFetcher {
	return func (path string) (Metadata, error) {
		m := Metadata{}

		resp, _ := httpClient.Get(fmt.Sprintf("%s/files/%s", filesApiUrl, path))
		if resp.StatusCode == http.StatusNotFound {
			return m, ErrFileNotRegistered
		}

		err := json.NewDecoder(resp.Body).Decode(&m)
		if err != nil {
			return Metadata{}, ErrBadJSONResponse
		}

		return m, nil
	}
}

func DownloadFile(s3client content.S3Client, vc content.VaultClient, vaultPath string) FileDownloader {
	return func(filePath string) (io.ReadCloser, error) {
		vp := vaultPath + "/" + filePath
		pskStr, err := vc.ReadKey(vp, VAULT_KEY)
		if err != nil {
			return nil, err
		}

		file, _, err := s3client.GetWithPSK(filePath, []byte(pskStr))

		return file, err
	}
}
