package steps

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	dprequest "github.com/ONSdigital/dp-net/v2/request"
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
	ctx.Step(`^I should receive the private file "([^"]*)"$`, d.iShouldReceiveThePrivateFile)
	ctx.Step(`^is not yet published$`, d.isNotYetPublished)
	ctx.Step(`^I download the file "([^"]*)"$`, d.iDownloadTheFile)
	ctx.Step(`^the file "([^"]*)" has the metadata:$`, d.theFileMetadata)
	ctx.Step(`^the application is in "([^"]*)" mode$`, d.weAreInWebMode)
	ctx.Step(`^the headers should be:$`, d.theHeadersShouldBe)
	ctx.Step(`^the file content should be:$`, d.theFileContentShouldBe)
	ctx.Step(`^the file "([^"]*)" has not been uploaded$`, d.theFileHasNotBeenUploaded)
	ctx.Step(`^the file "([^"]*)" is in S3 with content:$`, d.theFileStoredInS3WithContent)
	ctx.Step(`^I should be redirected to "([^"]*)"$`, d.iShouldBeRedirectedTo)
	ctx.Step(`^the GET request with path \("([^"]*)"\) should contain an authorization header containing "([^"]*)"$`, d.theGETRequestWithPathShouldContainAnAuthorizationHeaderContaining)

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

func (d *DownloadServiceComponent) iShouldReceiveThePrivateFile(filename string) error {
	assert.Equal(d.ApiFeature, http.StatusOK, d.ApiFeature.HttpResponse.StatusCode)
	assert.Equal(d.ApiFeature, "attachment; filename="+filename, d.ApiFeature.HttpResponse.Header.Get("Content-Disposition"))

	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) isNotYetPublished() error {
	return nil
}

func (d *DownloadServiceComponent) theFileHasNotBeenUploaded(filename string) error {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)

	}))

	d.cfg.FilesApiURL = server.URL

	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) iDownloadTheFile(filepath string) error {
	return d.ApiFeature.IGet(fmt.Sprintf("/downloads-new/%s", filepath))
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

// here
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

	assert.Equal(d.ApiFeature, url, d.ApiFeature.HttpResponse.Header.Get("Location"))

	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) theGETRequestWithPathShouldContainAnAuthorizationHeaderContaining(filepath, expectedAuthHeader string) error {
	assert.Equal(d.ApiFeature, expectedAuthHeader, requests[fmt.Sprintf("/files/%s", filepath)])
	return d.ApiFeature.StepError()
}
