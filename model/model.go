package model

import (
	"context"

	"github.com/google/uuid"

	"github.com/ONSdigital/dp-download-service/storage"
)

// Generate mocks of dependencies
//
//go:generate moq -rm -pkg model_test -out moq_storage_test.go . Storage

// Storage describes what we expect our underlying storage layer to implement.
//
type Storage interface {
	CreateDataset(ctx context.Context, document *storage.DatasetDocument) error
}

// A DatasetDocument is saved to storage.
// Currently identical to what is passed to the storage layer,
// but may be expanded later.
//
type DatasetDocument storage.DatasetDocument

// Model implements business logic and calls the storage layer.
//
type Model struct {
	storage Storage
}

// New returns a new Model using storage as its underlying storage layer.
//
func New(storage Storage) *Model {
	return &Model{
		storage: storage,
	}
}

// Create verifies and creates a new dataset document.
//
func (m *Model) Create(ctx context.Context, document *DatasetDocument) (string, error) {
	// do parameter and business validation here

	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	document.ID = id.String()

	err = m.storage.CreateDataset(ctx, (*storage.DatasetDocument)(document))
	if err != nil {
		return "", err
	}
	return document.ID, err
}
