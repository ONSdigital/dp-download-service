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
}

func newFakeFilesApiHttpClient(statusCode int, body string) HTTPClient {
	return &fakeHttpClient{
		statusCode: statusCode,
		body:       body,
	}
}

func (f fakeHttpClient) Do(ctx context.Context, req *http.Request) (resp *http.Response, err error) {
	return &http.Response{
		StatusCode: f.statusCode,
		Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(f.body))),
	}, nil
}

func (s *RetrieverTestSuite) TestReturnsBadJSONResponseWhenCannotParseJSON() {

	fhc := newFakeFilesApiHttpClient(200, "{bad json")

	_, err := FetchMetadata("", fhc)(context.Background(), "data/file.csv")

	s.Equal(ErrBadJSONResponse, err)
}

func (s *RetrieverTestSuite) TestReturns403ResponseWhenFilesResponds403() {

	fhc := newFakeFilesApiHttpClient(403, `{"errors": [{"errorCode": "403", "description": "Not Authorized"}]}`)

	_, err := FetchMetadata("", fhc)(context.Background(), "data/file.csv")

	s.Equal(ErrNotAuthorised, err)
}

func (s *RetrieverTestSuite) TestReturns500ResponseWhenFilesResponds500() {
	fhc := newFakeFilesApiHttpClient(500, `{"errors": [{"errorCode": "500", "description": "Internal Server Error"}]}`)

	_, err := FetchMetadata("", fhc)(context.Background(), "data/file.csv")

	s.Equal(ErrInternalServerError, err)
}

func (s *RetrieverTestSuite) TestReturnsUnknownErrorWhenFilesResponds418() {
	fhc := newFakeFilesApiHttpClient(418, `{"errors": [{"errorCode": "418", "description": "I'm a Teapot"}]}`)

	_, err := FetchMetadata("", fhc)(context.Background(), "data/file.csv")

	s.Equal(ErrUnknown, err)
}

func (s *RetrieverTestSuite) TestFetchMetadata() {
	filePath := "data/file.csv"

	fhc := newFakeFilesApiHttpClient(200, "{}")

	metadata, err := FetchMetadata("", fhc)(context.Background(), filePath)

	s.NoError(err)
	s.Equal(Metadata{}, metadata)
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
