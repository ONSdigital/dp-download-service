package files

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ONSdigital/dp-download-service/content"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
)

type HTTPClient interface {
	Get(url string) (resp *http.Response, err error)
}

var ErrFileNotRegistered = errors.New("file not registered")
var ErrBadJSONResponse = errors.New("could not decode JSON response from files api")

type Metadata struct {
	Path          string `json:"path"`
	IsPublishable *bool  `json:"is_publishable,omitempty"`
	CollectionID  string `json:"collection_id"`
	Title         string `json:"title"`
	SizeInBytes   uint64 `json:"size_in_bytes"`
	Type          string `json:"type"`
	Licence       string `json:"licence"`
	LicenceUrl    string `json:"licence_url"`
}

func (m Metadata) GetFilename() string {
	return filepath.Base(m.Path)
}

func (m Metadata) GetContentLength() string {
	return strconv.FormatUint(m.SizeInBytes, 10)
}

type FileRetriever func(path string) (Metadata, io.ReadCloser, error)

type Store struct {
	s3c         content.S3Client
	filesApiUrl string
	httpClient  HTTPClient
}

func NewStore(filesApiUrl string, s3client content.S3Client, httpClient HTTPClient) Store {
	return Store{s3client, filesApiUrl, httpClient}
}

func (s Store) RetrieveBy(path string) (Metadata, io.ReadCloser, error) {
	m := Metadata{}

	resp, _ := s.httpClient.Get(fmt.Sprintf("%s/v1/files/%s", s.filesApiUrl, path))
	if resp.StatusCode == http.StatusNotFound {
		return m, nil, ErrFileNotRegistered
	}

	err := json.NewDecoder(resp.Body).Decode(&m)
	if err != nil {
		return Metadata{}, nil, ErrBadJSONResponse
	}

	file, _, err := s.s3c.Get(path)

	return m, file, err
}
