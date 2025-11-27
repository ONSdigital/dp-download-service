package steps

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	componenttest "github.com/ONSdigital/dp-component-test"
	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-download-service/service"
	filesAPIModels "github.com/ONSdigital/dp-files-api/files"
	dphttp "github.com/ONSdigital/dp-net/v3/http"
	"github.com/ONSdigital/log.go/v2/log"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	localStackHost = "http://localstack:4566"
)

type DownloadServiceComponent struct {
	DpHttpServer *dphttp.Server
	svc          *service.Download
	ApiFeature   *componenttest.APIFeature
	errChan      chan error
	cfg          *config.Config
	deps         *External
}

func NewDownloadServiceComponent(fake_auth_url string) *DownloadServiceComponent {
	//os.Setenv("ZEBEDEE_URL", fake_auth_url)
	s := dphttp.NewServer("", http.NewServeMux())
	s.HandleOSSignals = false

	d := &DownloadServiceComponent{
		DpHttpServer: s,
		errChan:      make(chan error),
	}

	os.Setenv("DATABASE_NAME", "testing")

	log.Namespace = "dp-download-service"

	os.Setenv("ZEBEDEE_URL", fake_auth_url)
	os.Setenv("PUBLIC_BUCKET_URL", "http://public-bucket.com/")
	os.Setenv("IS_PUBLISHING", "false")

	d.cfg, _ = config.Get()

	d.deps = &External{Server: d.DpHttpServer}

	return d
}

func (d *DownloadServiceComponent) Initialiser() (http.Handler, error) {
	d.svc, _ = service.New(context.Background(), "1", "1", "1", d.cfg, d.deps)
	d.svc.Run(context.Background())
	time.Sleep(5 * time.Second)

	return d.DpHttpServer.Handler, nil
}

func (d *DownloadServiceComponent) Reset() {
	// Clear file events from previous scenarios
	d.deps.CreatedFileEvents = []filesAPIModels.FileEvent{}

	cfg, _ := config.Get()
	ctx := context.Background()

	// clear out test bucket
	awsConfig, err := awsConfig.LoadDefaultConfig(ctx,
		awsConfig.WithRegion(cfg.AwsRegion),
		awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to create aws config: %s", err.Error()))
	}

	s3client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(localStackHost)
		o.UsePathStyle = true
	})

	objectsToDelete, err := s3client.ListObjects(ctx, &s3.ListObjectsInput{
		Bucket: aws.String(cfg.BucketName),
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to list objects in localstack s3: %s", err.Error()))
	}

	for _, object := range objectsToDelete.Contents {
		deleteObjectInput := &s3.DeleteObjectInput{
			Bucket: aws.String(cfg.BucketName),
			Key:    object.Key,
		}
		_, err = s3client.DeleteObject(ctx, deleteObjectInput)
		if err != nil {
			panic(fmt.Sprintf("Failed to delete object in localstack s3: %s", err.Error()))
		}
	}

	if err != nil {
		panic(fmt.Sprintf("Failed to delete objects in localstack s3: %s", err.Error()))
	}
}

func (d *DownloadServiceComponent) Close() error {
	if d.svc != nil {
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		return d.svc.Close(ctx)
	}

	return nil
}
