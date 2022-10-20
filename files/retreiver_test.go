package files

import (
	"bytes"
	"context"
	"encoding/hex"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/ONSdigital/dp-download-service/content/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

type RetrieverTestSuite struct {
	suite.Suite
	s3c *mocks.MockS3Client
	vc  *mocks.MockVaultClient
}

func (s *RetrieverTestSuite) SetupTest() {
	s.s3c = mocks.NewMockS3Client(gomock.NewController(s.T()))
	s.vc = mocks.NewMockVaultClient(gomock.NewController(s.T()))
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
		Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(f.body))),
	}, f.err
}

func (s *RetrieverTestSuite) TestDownloadFile() {
	filePath := "data/file.csv"
	psk := "123456789123456789"
	encryptionKey, _ := hex.DecodeString(psk)

	fileContent := ioutil.NopCloser(bytes.NewBuffer([]byte("file content")))

	s.s3c.EXPECT().GetWithPSK(filePath, encryptionKey).Return(fileContent, nil, nil)

	s.vc.EXPECT().ReadKey("/"+filePath, VAULT_KEY).Return(psk, nil)

	file, err := DownloadFile(s.s3c, s.vc, "")(filePath)

	s.NoError(err)
	s.Equal(fileContent, file)
}

func (s *RetrieverTestSuite) TestDownloadFileEncyptionKeyContainNonHexCharacter() {
	filePath := "data/file.csv"
	psk := "NON HEX CHARACTERS"

	s.vc.EXPECT().ReadKey("/"+filePath, VAULT_KEY).Return(psk, nil)

	_, err := DownloadFile(s.s3c, s.vc, "")(filePath)

	s.Error(err)
}
