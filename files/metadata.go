package files

import (
	"path/filepath"
	"strconv"

	filesSDK "github.com/ONSdigital/dp-files-api/files"
)

const (
	CREATED   string = "CREATED"   // first chunk uploaded
	UPLOADED  string = "UPLOADED"  // all chunks uploaded
	PUBLISHED string = "PUBLISHED" // published - authorized for public download
	MOVED     string = "MOVED"     // available from S3/CDN
)

func GetFilename(m *filesSDK.StoredRegisteredMetaData) string {
	return filepath.Base(m.Path)
}

func GetContentLength(m *filesSDK.StoredRegisteredMetaData) string {
	return strconv.FormatUint(m.SizeInBytes, 10)
}

func Unpublished(m *filesSDK.StoredRegisteredMetaData) bool {
	return !(m.State == PUBLISHED || m.State == MOVED)
}

func UploadIncomplete(m *filesSDK.StoredRegisteredMetaData) bool {
	return m.State == CREATED
}

func Moved(m *filesSDK.StoredRegisteredMetaData) bool {
	return m.State == MOVED
}

func Uploaded(m *filesSDK.StoredRegisteredMetaData) bool {
	return m.State == UPLOADED
}
