package files

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/ONSdigital/dp-download-service/content/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

type RetrieverTestSuite struct {
	suite.Suite
	s3c *mocks.MockS3Client
}

func (s *RetrieverTestSuite) SetupTest() {
	s.s3c = mocks.NewMockS3Client(gomock.NewController(s.T()))
}

func TestRetrieverTestSuite(t *testing.T) {
	suite.Run(t, new(RetrieverTestSuite))
}

func (s *RetrieverTestSuite) TestDownloadFile() {
	filePath := "data/file.csv"

	fileContent := io.NopCloser(bytes.NewBuffer([]byte("file content")))

	s.s3c.EXPECT().Get(gomock.Any(), filePath).Return(fileContent, nil, nil)

	file, err := DownloadFile(context.Background(), s.s3c)(filePath)

	s.NoError(err)
	s.Equal(fileContent, file)
}
