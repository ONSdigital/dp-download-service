package files_test

import (
	"github.com/ONSdigital/dp-download-service/files"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUnpublished(t *testing.T) {
	testFiles := []struct {
		State               files.State
		ExpectedUnpublished bool
	}{
		{State: files.CREATED, ExpectedUnpublished: true},
		{State: files.UPLOADED, ExpectedUnpublished: true},
		{State: files.PUBLISHED, ExpectedUnpublished: false},
		{State: files.DECRYPTED, ExpectedUnpublished: false},
	}

	for _, file := range testFiles {
		metadata := files.Metadata{
			State: file.State,
		}

		assert.Equal(t, file.ExpectedUnpublished, metadata.Unpublished())
	}
}
