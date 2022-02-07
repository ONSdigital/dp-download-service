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

type FileRetriever func(path string) (Metadata, io.ReadCloser, error)

type Store struct {
	s3c         content.S3Client
	filesApiUrl string
	httpClient  HTTPClient
	vaultClient content.VaultClient
	vaultPath   string
}

func NewStore(filesApiUrl string, s3client content.S3Client, httpClient HTTPClient, vc content.VaultClient, vaultPath string) Store {
	return Store{s3client, filesApiUrl, httpClient, vc, vaultPath}
}

func (s Store) RetrieveBy(filePath string) (Metadata, io.ReadCloser, error) {
	metadata, err := s.fetchMetadata(filePath)
	if err != nil {
		return Metadata{}, nil, err
	}

	file, err := s.downloadFile(filePath)
	if err != nil {
		return Metadata{}, nil, err
	}

	return metadata, file, nil
}

func (s Store) fetchMetadata(path string) (Metadata, error) {
	m := Metadata{}

	resp, _ := s.httpClient.Get(fmt.Sprintf("%s/v1/files/%s", s.filesApiUrl, path))
	if resp.StatusCode == http.StatusNotFound {
		return m, ErrFileNotRegistered
	}

	err := json.NewDecoder(resp.Body).Decode(&m)
	if err != nil {
		return Metadata{}, ErrBadJSONResponse
	}

	return m, nil
}

func (s Store) downloadFile(filePath string) (io.ReadCloser, error) {
	vp := s.vaultPath + "/" + filePath
	pskStr, err := s.vaultClient.ReadKey(vp, VAULT_KEY)
	if err != nil {
		return nil, err
	}

	file, _, err := s.s3c.GetWithPSK(filePath, []byte(pskStr))

	return file, err
}
