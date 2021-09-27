package mongo

import (
	"context"

	"github.com/ONSdigital/dp-download-service/storage"
)

func (m *Mongo) CreateDataset(ctx context.Context, document *storage.DatasetDocument) error {
	_, err := m.datasetCollection.InsertOne(ctx, document)
	return err
}
