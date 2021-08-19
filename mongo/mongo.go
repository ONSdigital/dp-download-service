package mongo

import (
	"context"
	"errors"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	dpMongoLock "github.com/ONSdigital/dp-mongodb/v2/dplock"
	dpMongoHealth "github.com/ONSdigital/dp-mongodb/v2/health"
	dpmongo "github.com/ONSdigital/dp-mongodb/v2/mongodb"
)

const (
	connectTimeoutInSeconds = 5
	queryTimeoutInSeconds   = 15
)

// Mongo represents a simplistic MongoDB configuration.
type Mongo struct {
	Username     string
	Password     string
	IsSSL        bool
	CodeListURL  string
	Collection   string
	Database     string
	DatasetURL   string
	Connection   *dpmongo.MongoConnection
	URI          string
	client       *dpMongoHealth.Client
	healthClient *dpMongoHealth.CheckMongoClient
	lockClient   *dpMongoLock.Lock
}

func (m *Mongo) Init(ctx context.Context) (err error) {
	if m.Connection != nil {
		return errors.New("mongo connection already exists")
	}

	connCfg := &dpmongo.MongoConnectionConfig{
		IsSSL:                   m.IsSSL,
		ConnectTimeoutInSeconds: connectTimeoutInSeconds,
		QueryTimeoutInSeconds:   queryTimeoutInSeconds,

		Username:                      m.Username,
		Password:                      m.Password,
		ClusterEndpoint:               m.URI,
		Database:                      m.Database,
		Collection:                    m.Collection,
		IsWriteConcernMajorityEnabled: true,
		IsStrongReadConcernEnabled:    false,
	}

	conn, err := dpmongo.Open(connCfg)
	if err != nil {
		return err
	}
	m.Connection = conn

	// set up databaseCollectionBuilder here when collections are known

	// Create client and healthclient from session
	m.client = dpMongoHealth.NewClientWithCollections(m.Connection, nil)
	m.healthClient = &dpMongoHealth.CheckMongoClient{
		Client:      *m.client,
		Healthcheck: m.client.Healthcheck,
	}

	// create lock client here when collections are known
	return nil
}

// Close represents mongo session closing within the context deadline
func (m *Mongo) Close(ctx context.Context) error {
	m.lockClient.Close(ctx)
	return m.Connection.Close(ctx)
}

// Checker is called by the healthcheck library to check the health state of this mongoDB instance
func (m *Mongo) Checker(ctx context.Context, state *healthcheck.CheckState) error {
	return m.healthClient.Checker(ctx, state)
}
