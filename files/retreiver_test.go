package files

import (
	"bytes"
	"context"
	"io"
	"net/http"
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

type fakeHttpClient struct {
	HTTPClient
	statusCode int
	body       string
	err        error
}

func newFakeFilesApiHttpClient(statusCode int, body string) HTTPClient {
	return &fakeHttpClient{
		statusCode: statusCode,
		body:       body,
	}
}

func newFakeFilesApiErroringHttpClient(err error) HTTPClient {
	return &fakeHttpClient{
		err: err,
	}
}

func (f fakeHttpClient) Do(ctx context.Context, req *http.Request) (resp *http.Response, err error) {
	return &http.Response{
		StatusCode: f.statusCode,
		Body:       io.NopCloser(bytes.NewBuffer([]byte(f.body))),
	}, f.err
}

func (s *RetrieverTestSuite) TestDownloadFile() {
	filePath := "data/file.csv"

	fileContent := io.NopCloser(bytes.NewBuffer([]byte("file content")))

	s.s3c.EXPECT().Get(gomock.Any(), filePath).Return(fileContent, nil, nil)

	file, err := DownloadFile(context.Background(), s.s3c)(filePath)

	s.NoError(err)
	s.Equal(fileContent, file)
}
