package mongo

import (
	"context"

	"github.com/ONSdigital/dp-download-service/storage"
	"github.com/ONSdigital/log.go/v2/log"
)

func (m *Mongo) CreateDataset(ctx context.Context, document *storage.DatasetDocument) error {
	logData := log.Data{
		"document": document,
	}
	log.Info(ctx, "would be creating dataset", logData)

	return nil
}
