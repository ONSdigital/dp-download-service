package mongo

import (
	"context"

	"github.com/ONSdigital/dp-download-service/config"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	dpMongoLock "github.com/ONSdigital/dp-mongodb/v2/dplock"
	dpMongoHealth "github.com/ONSdigital/dp-mongodb/v2/health"
	dpmongo "github.com/ONSdigital/dp-mongodb/v2/mongodb"
)

const (
	connectTimeoutInSeconds = 5
	queryTimeoutInSeconds   = 15

	datasetCollection = "datasets" // name of collection holding dataset documents
	locksCollection   = "locks"    // name of collection holding locks
)

// Mongo is a concrete storage layer using docdb or mongo.
//
// The dp-mongodb's Mongo abstraction has a distinguished
// collection (.Collection in MongoConnectionConfig{}), which
// seems to be used as a sort of 'default' collection for
// simple applications, and it is also the collection used by
// dplock.
//
// Our application requires multiple collections, so we use
// the distinguished collection for locks, and separate
// collections for application storage.
//
// And we keep the collection handles in the Mongo struct
// below instead of collection names.
// Saves us creating a new handle on every operation.
// The mongo driver docs confirm handles are safe for
// concurrent use.
// https://pkg.go.dev/go.mongodb.org/mongo-driver@v1.7.2/mongo#Collection
//
type Mongo struct {
	datasetURL        string
	database          string
	datasetCollection *dpmongo.Collection // handle for dataset collection
	connection        *dpmongo.MongoConnection
	uri               string
	client            *dpMongoHealth.Client
	healthClient      *dpMongoHealth.CheckMongoClient
	lockClient        *dpMongoLock.Lock
}

func New(ctx context.Context, cfg *config.Config) (*Mongo, error) {
	m := &Mongo{
		datasetURL: cfg.DatasetAPIURL,
		uri:        cfg.MongoConfig.BindAddr,
		database:   cfg.MongoConfig.Database,
	}

	connCfg := &dpmongo.MongoConnectionConfig{
		IsSSL:                   cfg.MongoConfig.IsSSL,
		ConnectTimeoutInSeconds: connectTimeoutInSeconds,
		QueryTimeoutInSeconds:   queryTimeoutInSeconds,

		Username:                      cfg.MongoConfig.Username,
		Password:                      cfg.MongoConfig.Password,
		ClusterEndpoint:               cfg.MongoConfig.BindAddr,
		Database:                      cfg.MongoConfig.Database,
		Collection:                    locksCollection,
		IsWriteConcernMajorityEnabled: true,
		IsStrongReadConcernEnabled:    false,
	}

	conn, err := dpmongo.Open(connCfg)
	if err != nil {
		return nil, err
	}
	m.connection = conn

	// itemise collections we want to monitor
	//
	monitoredCollections := make(map[dpMongoHealth.Database][]dpMongoHealth.Collection)
	monitoredCollections[(dpMongoHealth.Database)(m.database)] = []dpMongoHealth.Collection{
		(dpMongoHealth.Collection)(datasetCollection),
		(dpMongoHealth.Collection)(locksCollection),
	}

	// Create client and healthclient from session
	//
	m.client = dpMongoHealth.NewClientWithCollections(m.connection, monitoredCollections)
	m.healthClient = &dpMongoHealth.CheckMongoClient{
		Client:      *m.client,
		Healthcheck: m.client.Healthcheck,
	}

	// create lock client
	//
	m.lockClient = dpMongoLock.New(ctx, m.connection, locksCollection)

	// create collection handles
	//
	m.datasetCollection = m.connection.C(datasetCollection)

	return m, nil
}

func (m *Mongo) URI() string {
	return m.uri
}

// Close represents mongo session closing within the context deadline
func (m *Mongo) Close(ctx context.Context) error {
	m.lockClient.Close(ctx)
	return m.connection.Close(ctx)
}

// Checker is called by the healthcheck library to check the health state of this mongoDB instance
func (m *Mongo) Checker(ctx context.Context, state *healthcheck.CheckState) error {
	return m.healthClient.Checker(ctx, state)
}
