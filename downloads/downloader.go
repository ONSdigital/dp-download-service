package downloads

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/ONSdigital/dp-api-clients-go/v2/dataset"
	"github.com/ONSdigital/dp-api-clients-go/v2/filter"
	"github.com/ONSdigital/dp-api-clients-go/v2/image"
	filesSDK "github.com/ONSdigital/dp-files-api/files"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/log.go/v2/log"
)

//go:generate mockgen -destination mocks/mocks.go -package mocks github.com/ONSdigital/dp-download-service/downloads FilterClient,DatasetClient,ImageClient,FilesClient

// FilterClient is an interface to represent methods called to action on the filter api
type FilterClient interface {
	GetOutput(ctx context.Context, userAuthToken, serviceAuthToken, downloadServiceToken, collectionID, filterOutputID string) (m filter.Model, err error)
	Checker(ctx context.Context, check *healthcheck.CheckState) error
}

// DatasetClient is an interface to represent methods called to action on the dataset api
type DatasetClient interface {
	GetVersion(ctx context.Context, userAuthToken, serviceAuthToken, downloadServiceAuthToken, collectionID, datasetID, edition, version string) (m dataset.Version, err error)
	GetInstance(ctx context.Context, userAuthToken, serviceAuthToken, collectionID, instanceID, ifMatch string) (m dataset.Instance, eTag string, err error)
	Checker(ctx context.Context, check *healthcheck.CheckState) error
}

// ImageClient is an interface to represent methods called to action on the image api
type ImageClient interface {
	GetDownloadVariant(ctx context.Context, userAuthToken, serviceAuthToken, collectionID, imageID, variant string) (m image.ImageDownload, err error)
	Checker(ctx context.Context, check *healthcheck.CheckState) error
}

// FilesClient is interface to the files api
type FilesClient interface {
	GetFile(ctx context.Context, path string) (*filesSDK.StoredRegisteredMetaData, error)
	CreateFileEvent(ctx context.Context, event filesSDK.FileEvent) (*filesSDK.FileEvent, error)
	Checker(ctx context.Context, state *healthcheck.CheckState) error
}

// FileType - iota enum of possible file types that can be download
type FileType int

// Possible values for a FileType of a download. It can only be one of the following:
const (
	TypeDatasetVersion FileType = iota
	TypeFilterOutput
	TypeImage
)

// Model is a struct that contains all the required information to download a file.
type Model struct {
	IsPublished     bool
	Public          string
	PrivateS3Path   string
	PrivateFilename string
}

// Parameters is the union of required parameters to perform all downloads
type Parameters struct {
	UserAuthToken        string
	ServiceAuthToken     string
	DownloadServiceToken string
	CollectionID         string
	FilterOutputID       string
	InstanceID           string
	DatasetID            string
	Edition              string
	Version              string
	ImageID              string
	Variant              string
	Filename             string
}

// Downloader is a struct that contains the clients to request metadata about the downloads
type Downloader struct {
	FilterCli  FilterClient
	DatasetCli DatasetClient
	ImageCli   ImageClient
}

// Get requests the required metadata using a client depending on the provided paramters
func (d Downloader) Get(ctx context.Context, p Parameters, fileType FileType, variant string) (Model, error) {
	switch fileType {

	case TypeImage:
		log.Info(ctx, "getting image downloads", log.Data{
			"image_id": p.ImageID,
			"variant":  p.Variant,
			"filename": p.Filename,
		})
		return d.getImageDownload(ctx, p, variant)

	case TypeFilterOutput:
		log.Info(ctx, "getting downloads for filter output job", log.Data{
			"filter_output_id": p.FilterOutputID,
			"collection_id":    p.CollectionID,
		})
		return d.getFilterOutputDownload(ctx, p, variant)

	case TypeDatasetVersion:
		log.Info(ctx, "getting downloads for dataset version", log.Data{
			"dataset_id":    p.DatasetID,
			"edition":       p.Edition,
			"version":       p.Version,
			"collection_id": p.CollectionID,
		})
		return d.getDatasetVersionDownload(ctx, p, variant)

	default:
		return Model{}, errors.New("unsupported file type")

	}
}

// getFilterOutputDownload gets the Model for a filter output job.
func (d Downloader) getFilterOutputDownload(ctx context.Context, p Parameters, variant string) (Model, error) {
	var downloads Model

	fo, err := d.FilterCli.GetOutput(ctx, p.UserAuthToken, p.ServiceAuthToken, p.DownloadServiceToken, p.CollectionID, p.FilterOutputID)

	if err != nil {
		return downloads, err
	}

	model := Model{
		IsPublished: fo.IsPublished,
	}

	v, ok := fo.Downloads[variant]
	if ok {
		// The filter output will be considered published (available for public downloads), when it is in 'published' state.
		model.Public = v.Public
		s3Path, filename, err := parseURL(v.Private)
		if err != nil {
			return downloads, err
		}
		model.PrivateS3Path = s3Path
		model.PrivateFilename = filename
	}
	return model, nil
}

// getDatasetVersionDownload gets the Model for a dataset version
func (d Downloader) getDatasetVersionDownload(ctx context.Context, p Parameters, variant string) (Model, error) {
	var downloads Model

	version, err := d.DatasetCli.GetVersion(ctx, p.UserAuthToken, p.ServiceAuthToken, p.DownloadServiceToken, p.CollectionID, p.DatasetID, p.Edition, p.Version)
	if err != nil {
		return downloads, err
	}

	model := Model{
		IsPublished: version.State == dataset.StatePublished.String(),
	}

	v, ok := version.Downloads[variant]
	if ok {
		// The dataset will be considered published (available for public downloads), when it is in 'published' state.
		model.Public = v.Public
		s3Path, filename, err := parseURL(v.Private)
		if err != nil {
			return downloads, err
		}
		model.PrivateS3Path = s3Path
		model.PrivateFilename = filename
	}

	return model, nil
}

// getImageDownload gets the Model for an image
func (d Downloader) getImageDownload(ctx context.Context, p Parameters, variant string) (Model, error) {
	var downloads Model

	imageVariant, err := d.ImageCli.GetDownloadVariant(ctx, p.UserAuthToken, p.ServiceAuthToken, p.CollectionID, p.ImageID, variant)
	if err != nil {
		return downloads, err
	}

	privatePath := fmt.Sprintf("images/%s/%s", p.ImageID, p.Variant)
	downloads = Model{
		// The variant will be considered published (available for public downloads), when it is in 'published' or 'completed' state.
		IsPublished:     ("published" == imageVariant.State || "completed" == imageVariant.State),
		PrivateS3Path:   privatePath,
		PrivateFilename: p.Filename,
	}
	if imageVariant.State == "completed" {
		downloads.Public = imageVariant.Href
	}

	return downloads, nil
}

// IsPublicLinkAvailable return true if public URI for the requested extension is available and the object is published
func (m Model) IsPublicLinkAvailable() bool {
	return len(m.Public) > 0 && m.IsPublished
}

func parseURL(urlString string) (path string, filename string, err error) {
	parsedUrl, err := url.Parse(urlString)
	if err != nil {
		return
	}
	path = strings.TrimLeft(parsedUrl.Path, "/")
	filename = filepath.Base(parsedUrl.Path)
	return
}
