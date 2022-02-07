package steps

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ONSdigital/dp-download-service/config"
	vault "github.com/ONSdigital/dp-vault"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/maxcnunes/httpfake"

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
	ctx.Step(`^the S3 file "([^"]*)" with content:$`, d.theS3FileWithContent)
	ctx.Step(`^we are in web mode$`, d.weAreInWebMode)
	ctx.Step(`^the headers should be:$`, d.theHeadersShouldBe)
	ctx.Step(`^the file content should be:$`, d.theFileContentShouldBe)
	ctx.Step(`^the file "([^"]*)" has not been uploaded$`, d.theFileHasNotBeenUploaded)
	ctx.Step(`^the file "([^"]*)" encrypted using key "([^"]*)" from Vault stored in S3 with content:$`, d.theFileEncryptedUsingKeyFromVaultStoredInSWithContent)

}

func (d *DownloadServiceComponent) weAreInWebMode() error {
	d.cfg.IsPublishing = false
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

	//return errors.New("BROKE")
	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) isNotYetPublished() error {
	return nil
}

func (d *DownloadServiceComponent) theFileHasNotBeenUploaded(filename string) error {
	server := httpfake.New()
	server.NewHandler().Get(fmt.Sprintf("/v1/files/%s", filename)).Reply(http.StatusNotFound).BodyString("")

	d.cfg.FilesApiURL = server.ResolveURL("")

	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) theFileHasBeenUploaded(filename string, metadata *godog.DocString) error {
	server := httpfake.New()
	server.NewHandler().Get(fmt.Sprintf("/v1/files/%s", filename)).Reply(http.StatusOK).BodyString(metadata.Content)

	d.cfg.FilesApiURL = server.ResolveURL("")

	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) iDownloadTheFile(filepath string) error {
	return d.ApiFeature.IGet(fmt.Sprintf("/v1/downloads/%s", filepath))
}

func (d *DownloadServiceComponent) theFileMetadata(filepath string, metadata *godog.DocString) error {
	server := httpfake.New()
	server.NewHandler().Get(fmt.Sprintf("/v1/files/%s", filepath)).Reply(http.StatusOK).BodyString(metadata.Content)

	d.cfg.FilesApiURL = server.ResolveURL("")

	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) theFileEncryptedUsingKeyFromVaultStoredInSWithContent(filepath string, encryptionkey string, content *godog.DocString) error {
	cfg, _ := config.Get()

	vaultPath := fmt.Sprintf("%s/%s", cfg.VaultPath, filepath)

	vaultClient, err := vault.CreateClient(cfg.VaultToken, cfg.VaultAddress, 1)
	if err != nil {
		return err
	}

	// store encryptionkey key in vault against the path <filepath>
	if err := vaultClient.WriteKey(vaultPath, "key", encryptionkey); err != nil {
		return err
	}

	//actualEncryptionKey, err := vaultClient.ReadKey(vaultPath, "key")
	c := bytes.NewBuffer([]byte(content.Content))

	// encrypt
	encryptedContent, err := encryptObjectContent([]byte(encryptionkey), c)
	if err != nil {
		fmt.Printf("encryption has failed: %v", err)
		panic("encryption has failed")
	}

	fmt.Printf("encrypted content : %v", encryptedContent)

	//storing
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
