package files

import (
	"path/filepath"
	"strconv"

	"github.com/ONSdigital/dp-api-clients-go/v2/files"
)

const (
	CREATED   string = "CREATED"   // first chunk uploaded
	UPLOADED  string = "UPLOADED"  // all chunks uploaded
	PUBLISHED string = "PUBLISHED" // published - authorized for public download
	DECRYPTED string = "DECRYPTED" // available decrypted from S3/CDN
)

func GetFilename(m *files.FileMetaData) string {
	return filepath.Base(m.Path)
}

func GetContentLength(m *files.FileMetaData) string {
	return strconv.FormatUint(m.SizeInBytes, 10)
}

func Unpublished(m *files.FileMetaData) bool {
	return !(m.State == PUBLISHED)
}

func UploadIncomplete(m *files.FileMetaData) bool {
	return m.State == CREATED
}

func Decrypted(m *files.FileMetaData) bool {
	return m.State == DECRYPTED
}

func Uploaded(m *files.FileMetaData) bool {
	return m.State == UPLOADED
}
