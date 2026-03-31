package steps

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/ONSdigital/dp-authorisation/v2/authorisation"
	"github.com/ONSdigital/dp-authorisation/v2/authorisationtest"
	dprequest "github.com/ONSdigital/dp-net/v3/request"
	permissionsAPISDK "github.com/ONSdigital/dp-permissions-api/sdk"
	s3client "github.com/ONSdigital/dp-s3/v3"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

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
	ctx.Step(`^I am an admin user accessing the file through a browser$`, d.iAmAnAdminUserAccessingTheFileThroughABrowser)
	ctx.Step(`^I am a viewer user with permission$`, d.viewerAllowedJWTToken)
	ctx.Step(`^I am a viewer user without permission$`, d.viewerNotAllowedJWTToken)
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

func (c *DownloadServiceComponent) viewerNotAllowedJWTToken() error {
	token, err := c.generateViewerAccessToken(
		"viewer2@ons.gov.uk",
		[]string{"role-viewer-not-allowed"},
	)
	if err != nil {
		return err
	}

	return c.ApiFeature.ISetTheHeaderTo("Authorization", "Bearer "+token)
}

func (c *DownloadServiceComponent) viewerAllowedJWTToken() error {
	token, err := c.generateViewerAccessToken(
		"viewer1@ons.gov.uk",
		[]string{"role-viewer-allowed"},
	)
	if err != nil {
		return err
	}

	return c.ApiFeature.ISetTheHeaderTo("Authorization", "Bearer "+token)
}

func (c *DownloadServiceComponent) generateViewerAccessToken(email string, groups []string) (string, error) {
	if err := c.ensureViewerKeys(); err != nil {
		return "", err
	}

	now := time.Now().Unix()

	claims := jwt.MapClaims{
		"sub":            "viewer-sub",
		"token_use":      "access",
		"auth_time":      now,
		"iss":            "https://cognito-idp.eu-west-2.amazonaws.com/eu-west-2_example",
		"exp":            now + 3600,
		"iat":            now,
		"client_id":      "component-test-client",
		"username":       email,
		"cognito:groups": groups,
	}

	t := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	t.Header["kid"] = c.viewerKID

	return t.SignedString(c.viewerPrivKey)
}

func (c *DownloadServiceComponent) ensureViewerKeys() error {
	if c.viewerPrivKey != nil && c.viewerKID != "" {
		return nil
	}

	// Generate RSA keypair
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generate viewer RSA key: %w", err)
	}
	if validationErr := priv.Validate(); validationErr != nil {
		return fmt.Errorf("validate viewer RSA key: %w", validationErr)
	}

	// Create kid
	kid := uuid.New().String()

	// Convert public key to PKIX DER and base64 encode
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return fmt.Errorf("marshal viewer public key: %w", err)
	}

	if c.cfg.AuthorisationConfig.JWTVerificationPublicKeys == nil {
		c.cfg.AuthorisationConfig.JWTVerificationPublicKeys = map[string]string{}
	}
	c.cfg.AuthorisationConfig.JWTVerificationPublicKeys[kid] = base64.StdEncoding.EncodeToString(pubDER)

	c.viewerPrivKey = priv
	c.viewerKID = kid
	return nil
}

func (c *DownloadServiceComponent) iAmAnAdminUserAccessingTheFileThroughABrowser() error {
	return c.ApiFeature.ISetTheHeaderTo("Cookie", "access_token="+authorisationtest.AdminJWTToken+";")
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
	if d.cfg.IsPublishing {
		d.cfg.AuthorisationConfig = authorisation.NewDefaultConfig()
		d.cfg.AuthorisationConfig.Enabled = true

		fakePermissionsAPI := setupFakePermissionsAPI()

		d.cfg.AuthorisationConfig.PermissionsAPIURL = fakePermissionsAPI.URL()
	}
	return nil
}

func setupFakePermissionsAPI() *authorisationtest.FakePermissionsAPI {
	fakePermissionsAPI := authorisationtest.NewFakePermissionsAPI()
	bundle := getPermissionsBundle()
	fakePermissionsAPI.Reset()
	if err := fakePermissionsAPI.UpdatePermissionsBundleResponse(bundle); err != nil {
		log.Error(context.Background(), "failed to update permissions bundle response", err)
	}
	return fakePermissionsAPI
}

func getPermissionsBundle() *permissionsAPISDK.Bundle {
	return &permissionsAPISDK.Bundle{
		"static-files:read": {
			"users/service": {
				{
					ID: "1",
				},
			},
			"groups/role-publisher": {
				{
					ID: "1",
				},
			},
			"groups/role-admin": {
				{
					ID: "1",
				},
			},
			"groups/role-viewer-allowed": {
				{
					ID: "1",
					Condition: permissionsAPISDK.Condition{
						Values:    []string{"cpih01/feb-2026"},
						Attribute: "dataset_edition",
						Operator:  "StringEquals",
					},
				},
			},
			"groups/role-viewer-not-allowed": {
				{
					ID: "1",
					Condition: permissionsAPISDK.Condition{
						Values:    []string{"1/45"},
						Attribute: "dataset_edition",
						Operator:  "StringEquals",
					},
				},
			},
		},
	}
}

func (d *DownloadServiceComponent) theHeadersShouldBe(expectedHeaders *godog.Table) error {
	headers, _ := assistdog.NewDefault().ParseMap(expectedHeaders)
	for key, value := range headers {
		err := d.ApiFeature.TheResponseHeaderShouldBe(key, value)
		if err != nil {
			return err
		}
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

	d.cfg.FilesAPIURL = server.URL

	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) theFileMetadata(filepath string, metadata *godog.DocString) error {
	requests = make(map[string]string)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path] = r.Header.Get(dprequest.AuthHeaderKey)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(metadata.Content))
		if err != nil {
			log.Error(context.Background(), "failed to write response", err)
		}
	}))
	d.cfg.FilesAPIURL = server.URL

	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) theFileStoredInS3WithContent(filepath string, content *godog.DocString) error {
	c := bytes.NewBuffer([]byte(content.Content))

	// store
	err := d.theS3FileWithContent(filepath, &godog.DocString{
		MediaType: "",
		Content:   c.String(),
	})
	assert.NoError(d.ApiFeature, err)

	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) theS3FileWithContent(filepath string, content *godog.DocString) error {
	cfg, _ := config.Get()
	ctx := context.Background()

	awsCfg, err := awsConfig.LoadDefaultConfig(ctx,
		awsConfig.WithRegion(cfg.AwsRegion),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	assert.NoError(d.ApiFeature, err)

	client := s3client.NewClientWithConfig(cfg.BucketName, awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(localStackHost)
		o.UsePathStyle = true
	})

	_, err = client.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(cfg.BucketName),
		Key:    aws.String(filepath),
		Body:   strings.NewReader(content.Content),
	})
	assert.NoError(d.ApiFeature, err)

	return d.ApiFeature.StepError()
}

func (d *DownloadServiceComponent) iShouldBeRedirectedTo(url string) error {
	err := d.ApiFeature.TheHTTPStatusCodeShouldBe("301")
	assert.NoError(d.ApiFeature, err)

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
