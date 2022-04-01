package steps

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/ONSdigital/dp-download-service/files"

	"github.com/ONSdigital/dp-download-service/config"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/cucumber/godog"
	"github.com/rdumont/assistdog"
	"github.com/stretchr/testify/assert"
)

func (d *DownloadServiceComponent) RegisterSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^I should receive the private file "([^"]*)"$`, d.iShouldReceiveThePrivateFile)
	ctx.Step(`^is not yet published$`, d.isNotYetPublished)
	ctx.Step(`^the file "([^"]*)" has been uploaded$`, d.theFileHasBeenUploaded)
	ctx.Step(`^I download the file "([^"]*)"$`, d.iDownloadTheFile)
	ctx.Step(`^the file "([^"]*)" metadata:$`, d.theFileMetadata)
	ctx.Step(`^the application is in "([^"]*)" mode$`, d.weAreInWebMode)
	ctx.Step(`^the headers should be:$`, d.theHeadersShouldBe)
	ctx.Step(`^the file content should be:$`, d.theFileContentShouldBe)
	ctx.Step(`^the file "([^"]*)" has not been uploaded$`, d.theFileHasNotBeenUploaded)
	ctx.Step(`^the file "([^"]*)" is encrypted in S3 with content:$`, d.theFileEncryptedUsingKeyFromVaultStoredInSWithContent)
	ctx.Step(`^I should be redirected to "([^"]*)"$`, d.iShouldBeRedirectedTo)

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

func (d *DownloadServiceComponent) theFileHasBeenUploaded(filename string, metadata *godog.DocString) error {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(metadata.Content))
	}))

	d.cfg.FilesApiURL = server.URL

	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) iDownloadTheFile(filepath string) error {
	return d.ApiFeature.IGet(fmt.Sprintf("/downloads-new/%s", filepath))
}

func (d *DownloadServiceComponent) theFileMetadata(filepath string, metadata *godog.DocString) error {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(metadata.Content))
	}))

	d.cfg.FilesApiURL = server.URL

	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) theFileEncryptedUsingKeyFromVaultStoredInSWithContent(filepath string, content *godog.DocString) error {
	cfg, _ := config.Get()

	vaultPath := fmt.Sprintf("%s/%s", cfg.VaultPath, filepath)

	encryptionKey := make([]byte, 16)
	rand.Read(encryptionKey)

	// store encryptionkey key in vault against the path <filepath>
	if err := d.vaultClient.WriteKey(vaultPath, files.VAULT_KEY, hex.EncodeToString(encryptionKey)); err != nil {
		return err
	}

	c := bytes.NewBuffer([]byte(content.Content))

	// encrypt
	encryptedContent, err := encryptObjectContent(encryptionKey, c)
	if err != nil {
		fmt.Printf("encryption has failed: %v", err)
		panic("encryption has failed")
	}

	// store
	d.theS3FileWithContent(filepath, &godog.DocString{
		MediaType: "",
		Content:   string(encryptedContent),
	})

	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) theS3FileWithContent(filepath string, content *godog.DocString) error {
	cfg, _ := config.Get()

	s, err := session.NewSession(&aws.Config{
		Endpoint:         aws.String(localStackHost),
		Region:           aws.String(cfg.AwsRegion),
		S3ForcePathStyle: aws.Bool(true),
		Credentials:      credentials.NewStaticCredentials("test", "test", ""),
	})
	assert.NoError(d.ApiFeature, err)

	_, err = s3manager.NewUploader(s).Upload(&s3manager.UploadInput{
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

func encryptObjectContent(psk []byte, b io.Reader) ([]byte, error) {
	unencryptedBytes, err := ioutil.ReadAll(b)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(psk)
	if err != nil {
		return nil, err
	}

	encryptedBytes := make([]byte, len(unencryptedBytes))

	stream := cipher.NewCFBEncrypter(block, psk)

	stream.XORKeyStream(encryptedBytes, unencryptedBytes)

	return encryptedBytes, nil
}
