package downloads

import (
	"context"

	"github.com/ONSdigital/dp-api-clients-go/dataset"
	"github.com/ONSdigital/dp-api-clients-go/filter"
	"github.com/ONSdigital/dp-api-clients-go/image"
	"github.com/ONSdigital/log.go/log"
)

//go:generate mockgen -destination mocks/mocks.go -package mocks github.com/ONSdigital/dp-download-service/downloads FilterClient,DatasetClient,ImageClient

// FilterClient is an interface to represent methods called to action on the filter api
type FilterClient interface {
	GetOutput(ctx context.Context, userAuthToken, serviceAuthToken, downloadServiceToken, collectionID, filterOutputID string) (m filter.Model, err error)
}

// DatasetClient is an interface to represent methods called to action on the dataset api
type DatasetClient interface {
	GetVersion(ctx context.Context, userAuthToken, serviceAuthToken, downloadServiceToken, collectionID, datasetID, edition, version string) (m dataset.Version, err error)
}

// ImageClient is an interface to represent methods called to action on the image api
type ImageClient interface {
	GetImage(ctx context.Context, userAuthToken, serviceAuthToken, collectionID, imageID string) (m image.Image, err error)
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
// Available is a map of available downloads for the model.
type Model struct {
	Available   map[string]Info
	IsPublished bool
}

// Info contains the public and private URLs to download file, of a generic type (filter, dataset version, image, ...)
type Info struct {
	Public  string
	Private string
}

// Parameters is the union of required paramters to perform all downloads
type Parameters struct {
	UserAuthToken        string
	ServiceAuthToken     string
	DownloadServiceToken string
	CollectionID         string
	FilterOutputID       string
	DatasetID            string
	Edition              string
	Version              string
	ImageID              string
	Variant              string
	Name                 string
	Ext                  string
}

// Downloader is a struct that contains the clients to request metadata about the downloads
type Downloader struct {
	FilterCli  FilterClient
	DatasetCli DatasetClient
	ImageCli   ImageClient
}

// Get requests the required metadata using a client depending on the provided paramters
func (d Downloader) Get(ctx context.Context, p Parameters, fileType FileType) (Model, error) {

	if fileType == TypeImage {
		log.Event(ctx, "getting image downloads", log.INFO, log.Data{
			"image_id": p.ImageID,
			"variant":  p.Variant,
			"name":     p.Name,
			"ext":      p.Ext,
		})
		return d.getImageDownloads(ctx, p)
	}

	if fileType == TypeFilterOutput {
		log.Event(ctx, "getting downloads for filter output job", log.INFO, log.Data{
			"filter_output_id": p.FilterOutputID,
			"collection_id":    p.CollectionID,
		})
		return d.getFilterOutputDownloads(ctx, p)
	}

	log.Event(ctx, "getting downloads for dataset version", log.INFO, log.Data{
		"dataset_id":    p.DatasetID,
		"edition":       p.Edition,
		"version":       p.Version,
		"collection_id": p.CollectionID,
	})
	return d.getDatasetVersionDownloads(ctx, p)
}

//getFilterOutputDownloads get the Model for a filter output job.
func (d Downloader) getFilterOutputDownloads(ctx context.Context, p Parameters) (Model, error) {
	var downloads Model

	fo, err := d.FilterCli.GetOutput(ctx, p.UserAuthToken, p.ServiceAuthToken, p.DownloadServiceToken, p.CollectionID, p.FilterOutputID)
	if err != nil {
		return downloads, err
	}

	// map filter outputs to Info structs
	available := make(map[string]Info)
	for k, v := range fo.Downloads {
		available[k] = Info{
			Private: v.Private,
			Public:  v.Public,
		}
	}

	// The filter output will be considered published (available for public downloads), when it is in 'published' state.
	return Model{
		IsPublished: fo.IsPublished,
		Available:   available,
	}, nil
}

//getDatasetVersionDownloads get the downloads for a dataset version
func (d Downloader) getDatasetVersionDownloads(ctx context.Context, p Parameters) (Model, error) {
	var downloads Model

	version, err := d.DatasetCli.GetVersion(ctx, p.UserAuthToken, p.ServiceAuthToken, p.DownloadServiceToken, p.CollectionID, p.DatasetID, p.Edition, p.Version)
	if err != nil {
		return downloads, err
	}

	// map dataset version to Info structs
	available := make(map[string]Info)
	for k, v := range version.Downloads {
		available[k] = Info{
			Public:  v.Public,
			Private: v.Private,
		}
	}

	// The dataset will be considered published (available for public downloads), when it is in 'published' state.
	return Model{
		IsPublished: "published" == version.State,
		Available:   available,
	}, nil
}

// getImageDownloads get the downloads for an image
func (d Downloader) getImageDownloads(ctx context.Context, p Parameters) (Model, error) {
	var downloads Model

	image, err := d.ImageCli.GetImage(ctx, p.UserAuthToken, p.ServiceAuthToken, p.CollectionID, p.ImageID)
	if err != nil {
		return downloads, err
	}

	// map image downloads to map of Info structs, assumign Href is the public URL when the image is published or completed
	available := make(map[string]Info)
	for k, v := range image.Downloads {
		available[k] = Info{
			Public:  v.Href,
			Private: v.Private,
		}
	}

	// The image will be considered published (available for public downloads), when it is in 'published' or 'completed' state.
	return Model{
		IsPublished: ("published" == image.State || "completed" == image.State),
		Available:   available,
	}, nil
}

// IsPublicLinkAvailable return true if public URI for the requested extension is available and the object is published
func (m Model) IsPublicLinkAvailable(variant string) bool {
	return len(m.Available[variant].Public) > 0 && m.IsPublished
}
