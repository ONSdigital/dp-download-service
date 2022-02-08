package files

import (
	"bytes"
	"github.com/ONSdigital/dp-download-service/content/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"io/ioutil"
	"net/http"
	"testing"
)

type RetrieverTestSuite struct {
	suite.Suite
	s3c *mocks.MockS3Client
	vc  *mocks.MockVaultClient
}

func (suite *RetrieverTestSuite) SetupTest() {
	suite.s3c = mocks.NewMockS3Client(gomock.NewController(suite.T()))
	suite.vc = mocks.NewMockVaultClient(gomock.NewController(suite.T()))
}

func TestRetrieverTestSuite(t *testing.T) {
	suite.Run(t, new(RetrieverTestSuite))
}

type fakeHttpClient struct {
	HTTPClient
	statusCode int
	body       string
}

func newFakeHttpClient(statusCode int, body string) HTTPClient {
	return &fakeHttpClient{
		statusCode: statusCode,
		body:       body,
	}
}

func (f fakeHttpClient) Get(url string) (resp *http.Response, err error) {
	return &http.Response{
		StatusCode: f.statusCode,
		Body:       ioutil.NopCloser(bytes.NewBuffer([]byte(f.body))),
	}, nil
}

func (suite *RetrieverTestSuite) TestReturnsBadJSONResponseWhenCannotParseJSON() {

	fhc := newFakeHttpClient(200, "{bad json")

	store := NewStore("", suite.s3c, fhc, nil, "")

	_, err := store.FetchMetadata("data/file.csv")

	assert.Equal(suite.T(), ErrBadJSONResponse, err)
}

func (suite *RetrieverTestSuite) TestFetchMetadata() {
	filePath := "data/file.csv"

	fhc := newFakeHttpClient(200, "{}")

	store := NewStore("", suite.s3c, fhc, suite.vc, "")

	metadata, err := store.FetchMetadata(filePath)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), Metadata{}, metadata)
}

func (suite *RetrieverTestSuite) TestDownloadFile() {
	filePath := "data/file.csv"
	psk := "psk"

	fileContent := ioutil.NopCloser(bytes.NewBuffer([]byte("file content")))

	suite.s3c.EXPECT().GetWithPSK(filePath, []byte(psk)).Return(fileContent, nil, nil)

	fhc := newFakeHttpClient(200, "{}")

	suite.vc.EXPECT().ReadKey("/"+filePath, VAULT_KEY).Return(psk, nil)

	store := NewStore("", suite.s3c, fhc, suite.vc, "")

	file, err := store.DownloadFile(filePath)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), fileContent, file)
}
