package service

// This set of methods is only available when testing so tests can
// access internal Download struct fields.

import (
	"time"

	"github.com/ONSdigital/dp-api-clients-go/v2/health"
	"github.com/ONSdigital/dp-download-service/content"
	"github.com/ONSdigital/dp-download-service/downloads"
)

func (svc *Download) GetDatasetClient() downloads.DatasetClient {
	return svc.datasetClient
}

func (svc *Download) GetFilterClient() downloads.FilterClient {
	return svc.filterClient
}

func (svc *Download) GetImageClient() downloads.ImageClient {
	return svc.imageClient
}

func (svc *Download) GetFilesClient() downloads.FilesClient {
	return svc.filesClient
}

func (svc *Download) GetS3Client() content.S3Client {
	return svc.s3Client
}

func (svc *Download) GetZebedeeHealthClient() *health.Client {
	return svc.zebedeeHealthClient
}

func (svc *Download) GetShutdownTimeout() time.Duration {
	return svc.shutdown
}

func (svc *Download) GetHealthChecker() HealthChecker {
	return svc.healthCheck
}
