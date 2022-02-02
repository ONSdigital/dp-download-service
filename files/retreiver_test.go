package files

import (
	"bytes"
	"github.com/ONSdigital/dp-download-service/content/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"testing"
)

type fakeHttpClient struct {
	HTTPClient
}

func (f fakeHttpClient) Get (url string) (resp *http.Response, err error) {
	return &http.Response{
		StatusCode: 200,
		Body: ioutil.NopCloser(bytes.NewBuffer([]byte("{ bad json "))),
	},nil
}

func TestReturnsBadJSONResponseWhenCannotParseJSON(t *testing.T) {
	mocks3c := mocks.NewMockS3Client(gomock.NewController(t))

	fhc := fakeHttpClient{}

	store := NewStore("", mocks3c, fhc)

	_, _, err := store.RetrieveBy("data/file.csv")

	assert.Equal(t, ErrBadJSONResponse, err)
}
