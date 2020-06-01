package downloads

import (
	"context"

	"github.com/ONSdigital/dp-api-clients-go/dataset"
	"github.com/ONSdigital/dp-api-clients-go/filter"
	"github.com/ONSdigital/log.go/log"
)

//go:generate mockgen -destination mocks/mocks.go -package mocks github.com/ONSdigital/dp-download-service/downloads FilterClient,DatasetClient

// FilterClient is an interface to represent methods called to action on the filter api
type FilterClient interface {
	GetOutput(ctx context.Context, userAuthToken, serviceAuthToken, downloadServiceToken, collectionID, filterOutputID string) (m filter.Model, err error)
}

// DatasetClient is an interface to represent methods called to action on the dataset api
type DatasetClient interface {
	GetVersion(ctx context.Context, userAuthToken, serviceAuthToken, downloadServiceToken, collectionID, datasetID, edition, version string) (m dataset.Version, err error)
}

type Model struct {
	Available   map[string]Info
	IsPublished bool
}

type Info struct {
	URL     string `json:"href"`
	Size    string `json:"size"`
	Public  string `json:"public,omitempty"`
	Private string `json:"private,omitempty"`
	Skipped bool   `json:"skipped,omitempty"`
}

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

type Downloader struct {
	FilterCli  FilterClient
	DatasetCli DatasetClient
}

func (d Downloader) Get(ctx context.Context, p Parameters) (Model, error) {
	if len(p.FilterOutputID) > 0 {
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

	mapping := make(map[string]Info)
	for k, v := range fo.Downloads {
		mapping[k] = Info(v)
	}

	downloads = Model{
		IsPublished: fo.IsPublished,
		Available:   mapping,
	}

	return downloads, nil
}

//getDatasetVersionDownloads get the downloads for a dataset version
func (d Downloader) getDatasetVersionDownloads(ctx context.Context, p Parameters) (Model, error) {
	var downloads Model

	version, err := d.DatasetCli.GetVersion(ctx, p.UserAuthToken, p.ServiceAuthToken, p.DownloadServiceToken, p.CollectionID, p.DatasetID, p.Edition, p.Version)
	if err != nil {
		return downloads, err
	}

	available := make(map[string]Info)
	for k, v := range version.Downloads {
		datasetDownloadWithSkipped := Info{
			URL:     v.URL,
			Size:    v.Size,
			Public:  v.Public,
			Private: v.Private,
			Skipped: false,
		}
		available[k] = datasetDownloadWithSkipped
	}

	downloads = Model{
		IsPublished: "published" == version.State,
		Available:   available,
	}

	return downloads, nil
}

// IsPublicLinkAvailable return true if public URI for the requested extension is available and the dataset is published
func (m Model) IsPublicLinkAvailable(extension string) bool {
	return len(m.Available[extension].Public) > 0 && m.IsPublished
}
