package downloads

import (
	"context"

	"github.com/ONSdigital/dp-api-clients-go/dataset"
	"github.com/ONSdigital/dp-api-clients-go/filter"
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
}

type Downloader struct {
	FilterCli  FilterClient
	DatasetCli DatasetClient
}

//GetFilterOutputDownloads get the Model for a filter output job.
func (d Downloader) GetFilterOutputDownloads(ctx context.Context, p Parameters) (Model, error) {
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

//GetDatasetVersionDownloads get the downloads for a dataset version
func (d Downloader) GetDatasetVersionDownloads(ctx context.Context, p Parameters) (Model, error) {
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
