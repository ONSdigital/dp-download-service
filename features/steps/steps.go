package steps

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	dprequest "github.com/ONSdigital/dp-net/v3/request"
	s3client "github.com/ONSdigital/dp-s3/v3"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/ONSdigital/dp-download-service/config"
	"github.com/cucumber/godog"
	"github.com/rdumont/assistdog"
	"github.com/stretchr/testify/assert"
)

var requests map[string]string

func (d *DownloadServiceComponent) RegisterSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^the file "([^"]*)" has the metadata:$`, d.theFileMetadata)
	ctx.Step(`^the application is in "([^"]*)" mode$`, d.weAreInWebMode)
	ctx.Step(`^the headers should be:$`, d.theHeadersShouldBe)
	ctx.Step(`^the file content should be:$`, d.theFileContentShouldBe)
	ctx.Step(`^the file "([^"]*)" has not been uploaded$`, d.theFileHasNotBeenUploaded)
	ctx.Step(`^the file "([^"]*)" is in S3 with content:$`, d.theFileStoredInS3WithContent)
	ctx.Step(`^I should be redirected to "([^"]*)"$`, d.iShouldBeRedirectedTo)
	ctx.Step(`^the response body should contain "([^"]*)"$`, d.theResponseBodyShouldContain)
	ctx.Step(`^the collection "([^"]*)" is marked as PUBLISHED$`, d.theCollectionIsMarkedAsPublished)
	ctx.Step(`^a file event with action "([^"]*)" and resource "([^"]*)" should be created by user "([^"]*)"$`, d.aFileEventShouldBeCreated)
	ctx.Step(`^no file event should be logged$`, d.noFileEventShouldBeLogged)
}

func (c *DownloadServiceComponent) theCollectionIsMarkedAsPublished(collectionID string) error {
	body := fmt.Sprintf(`{"state": %q}`, "PUBLISHED")
	err := c.ApiFeature.IPatch(
		fmt.Sprintf("/collection/%s", collectionID),
		&godog.DocString{
			MediaType: "application/json",
			Content:   body,
		},
	)
	assert.NoError(c.ApiFeature, err)
	return c.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) theResponseBodyShouldContain(expected string) error {
	bodyBytes, err := io.ReadAll(d.ApiFeature.HTTPResponse.Body)
	if err != nil {
		return err
	}
	body := string(bodyBytes)

	assert.Contains(d.ApiFeature, body, expected, "expected response body to contain %q but got %q", expected, body)

	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) weAreInWebMode(mode string) error {
	d.cfg.IsPublishing = mode == "publishing"
	return nil
}

func (d *DownloadServiceComponent) theHeadersShouldBe(expectedHeaders *godog.Table) error {
	headers, _ := assistdog.NewDefault().ParseMap(expectedHeaders)
	for key, value := range headers {
		d.ApiFeature.TheResponseHeaderShouldBe(key, value)
	}

	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) theFileContentShouldBe(expectedContent *godog.DocString) error {
	return d.ApiFeature.IShouldReceiveTheFollowingResponse(expectedContent)
}

func (d *DownloadServiceComponent) theFileHasNotBeenUploaded(filename string) error {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	d.cfg.FilesApiURL = server.URL

	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) theFileMetadata(filepath string, metadata *godog.DocString) error {
	requests = make(map[string]string)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path] = r.Header.Get(dprequest.AuthHeaderKey)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(metadata.Content))
	}))

	d.cfg.FilesApiURL = server.URL

	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) theFileStoredInS3WithContent(filepath string, content *godog.DocString) error {
	c := bytes.NewBuffer([]byte(content.Content))

	// store
	d.theS3FileWithContent(filepath, &godog.DocString{
		MediaType: "",
		Content:   c.String(),
	})

	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) theS3FileWithContent(filepath string, content *godog.DocString) error {
	cfg, _ := config.Get()
	ctx := context.Background()

	awsConfig, err := awsConfig.LoadDefaultConfig(ctx,
		awsConfig.WithRegion(cfg.AwsRegion),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	assert.NoError(d.ApiFeature, err)

	s3client := s3client.NewClientWithConfig(cfg.BucketName, awsConfig, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(localStackHost)
		o.UsePathStyle = true
	})

	_, err = s3client.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(cfg.BucketName),
		Key:    aws.String(filepath),
		Body:   strings.NewReader(content.Content),
	})
	assert.NoError(d.ApiFeature, err)

	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) iShouldBeRedirectedTo(url string) error {
	d.ApiFeature.TheHTTPStatusCodeShouldBe("301")

	assert.Equal(d.ApiFeature, url, d.ApiFeature.HTTPResponse.Header.Get("Location"))

	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) aFileEventShouldBeCreated(expectedAction, expectedResource, expectedUser string) error {
	assert.NotEmpty(d.ApiFeature, d.deps.CreatedFileEvents, "no file events were created")

	if len(d.deps.CreatedFileEvents) == 0 {
		return d.ApiFeature.StepError()
	}

	// Find matching event
	found := false
	for _, event := range d.deps.CreatedFileEvents {
		if event.File != nil &&
			event.File.Path == expectedResource &&
			event.Action == expectedAction &&
			event.RequestedBy != nil &&
			event.RequestedBy.Email == expectedUser {
			found = true
			break
		}
	}

	assert.True(d.ApiFeature, found, fmt.Sprintf(
		"expected file event with action=%s, resource=%s, user=%s but not found",
		expectedAction, expectedResource, expectedUser))

	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) noFileEventShouldBeLogged() error {

	assert.Empty(d.ApiFeature, d.deps.CreatedFileEvents, "expected no file events but found some")
	return d.ApiFeature.StepError()
}
