package files

import (
	"encoding/json"
	"fmt"
	"github.com/ONSdigital/dp-download-service/content"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
)

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
	s3c content.S3Client
	filesApiUrl string
}

func NewStore(filesApiUrl string, s3client content.S3Client) Store {
	return Store{s3client, filesApiUrl}
}

func (s Store) RetrieveBy(path string) (Metadata, io.ReadCloser, error) {
	m := Metadata{}
	resp, _ := http.Get(fmt.Sprintf("%s/v1/files/%s", s.filesApiUrl, path))
	json.NewDecoder(resp.Body).Decode(&m)

	file, _, _ := s.s3c.Get(path)

	return m, file, nil
}