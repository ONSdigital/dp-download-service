package files_test

import (
	"testing"

	fclient "github.com/ONSdigital/dp-api-clients-go/v2/files"
	"github.com/ONSdigital/dp-download-service/files"
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
	}

	for _, file := range testFiles {
		m := fclient.FileMetaData{
			State: file.State,
		}

		assert.Equal(t, file.ExpectedUnpublished, !(m.State == files.PUBLISHED))
	}
}
