package files_test

import (
	"testing"

	"github.com/ONSdigital/dp-download-service/files"
	filesAPIModels "github.com/ONSdigital/dp-files-api/files"
	"github.com/stretchr/testify/assert"
)

func TestUnpublished(t *testing.T) {
	testFiles := []struct {
		State               string
		ExpectedUnpublished bool
	}{
		{State: files.CREATED, ExpectedUnpublished: true},
		{State: files.UPLOADED, ExpectedUnpublished: true},
		{State: files.PUBLISHED, ExpectedUnpublished: false},
		{State: files.MOVED, ExpectedUnpublished: false},
	}

	for _, file := range testFiles {
		m := filesAPIModels.FileMetaData{
			State: file.State,
		}

		assert.Equal(t, file.ExpectedUnpublished, !(m.State == files.PUBLISHED || m.State == files.MOVED))
	}
}
