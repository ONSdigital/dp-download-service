package dataset

import (
	"context"

	"github.com/ONSdigital/dp-api-clients-go/dataset"
	"github.com/ONSdigital/dp-api-clients-go/filter"
)

//go:generate mockgen -destination mocks/mocks.go -package mocks github.com/ONSdigital/dp-download-service/dataset FilterClient,DatasetClient

// FilterClient is an interface to represent methods called to action on the filter api
type FilterClient interface {
	GetOutput(ctx context.Context, userAuthToken, serviceAuthToken, downloadServiceToken, collectionID, filterOutputID string) (m filter.Model, err error)
}

// DatasetClient is an interface to represent methods called to action on the dataset api
type DatasetClient interface {
	GetVersion(ctx context.Context, userAuthToken, serviceAuthToken, downloadServiceToken, collectionID, datasetID, edition, version string) (m dataset.Version, err error)
}

type Downloads struct {
	Available   map[string]DownloadInfo
	IsPublished bool
}

type DownloadInfo struct {
	URL     string `json:"href"`
	Size    string `json:"size"`
	Public  string `json:"public,omitempty"`
	Private string `json:"private,omitempty"`
	Skipped bool   `json:"skipped,omitempty"`
}

type Parameters struct {
	userAuthToken        string
	serviceAuthToken     string
	downloadServiceToken string
	collectionID         string
	filterOutputID       string
	datasetID            string
	edition              string
	version              string
}

type Downloader struct {
	FilterCli  FilterClient
	DatasetCli DatasetClient
}

//GetFilterOutputDownloads get the Downloads for a filter output job.
func (d Downloader) GetFilterOutputDownloads(ctx context.Context, p Parameters) (Downloads, error) {
	var downloads Downloads

	fo, err := d.FilterCli.GetOutput(ctx, p.userAuthToken, p.serviceAuthToken, p.downloadServiceToken, p.collectionID, p.filterOutputID)
	if err != nil {
		return downloads, err
	}

	mapping := make(map[string]DownloadInfo)
	for k, v := range fo.Downloads {
		mapping[k] = DownloadInfo(v)
	}

	downloads = Downloads{
		IsPublished: fo.IsPublished,
		Available:   mapping,
	}

	return downloads, nil
}

//GetDatasetVersionDownloads get the downloads for a dataset version
func (d Downloader) GetDatasetVersionDownloads(ctx context.Context, p Parameters) (Downloads, error) {
	var downloads Downloads

	version, err := d.DatasetCli.GetVersion(ctx, p.userAuthToken, p.serviceAuthToken, p.downloadServiceToken, p.collectionID, p.datasetID, p.edition, p.version)
	if err != nil {
		return downloads, err
	}

	available := make(map[string]DownloadInfo)
	for k, v := range version.Downloads {
		datasetDownloadWithSkipped := DownloadInfo{
			URL:     v.URL,
			Size:    v.Size,
			Public:  v.Public,
			Private: v.Private,
			Skipped: false,
		}
		available[k] = datasetDownloadWithSkipped
	}

	downloads = Downloads{
		IsPublished: "published" == version.State,
		Available:   available,
	}

	return downloads, nil
}
