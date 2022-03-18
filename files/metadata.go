package files

import (
	"path/filepath"
	"strconv"
)

const (
	UPLOADED  State = "UPLOADED"
	CREATED   State = "CREATED"
	PUBLISHED State = "PUBLISHED"
	DECRYPTED State = "DECRYPTED"
)

type State string

type Metadata struct {
	Path          string `json:"path"`
	IsPublishable *bool  `json:"is_publishable,omitempty"`
	CollectionID  string `json:"collection_id"`
	Title         string `json:"title"`
	SizeInBytes   uint64 `json:"size_in_bytes"`
	Type          string `json:"type"`
	Licence       string `json:"licence"`
	LicenceUrl    string `json:"licence_url"`
	State         State  `json:"state"`
}

func (m Metadata) GetFilename() string {
	return filepath.Base(m.Path)
}

func (m Metadata) GetContentLength() string {
	return strconv.FormatUint(m.SizeInBytes, 10)
}

func (m Metadata) Unpublished() bool {
	//return m.State == UPLOADED || m.State == CREATED
	return !(m.State == PUBLISHED || m.State == DECRYPTED)
}

func (m Metadata) Decrypted() bool {
	return m.State == DECRYPTED
}

func (m Metadata) Uploaded() bool {
	return m.State == UPLOADED
}
