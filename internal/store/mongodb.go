package store

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/halamri/go-dispatcher/internal/config"
)

// MongoDB wraps the MongoDB client. Database/collection are resolved from company_schema and data_source_name (deployment-specific).
type MongoDB struct {
	Client *mongo.Client
	cfg    *config.Config
}

// NewMongoDB creates a MongoDB client from config.
func NewMongoDB(ctx context.Context, cfg *config.Config) (*MongoDB, error) {
	uri := cfg.MongoDBURI
	if uri == "" {
		uri = fmt.Sprintf("mongodb://%s:%s", cfg.MongoDBHost, cfg.MongoDBPort)
		if cfg.MongoDBOptions != "" {
			uri += "?" + cfg.MongoDBOptions
		}
	}
	opts := options.Client().ApplyURI(uri)
	if cfg.MongoDBUser != "" && cfg.MongoDBURI == "" {
		opts.SetAuth(options.Credential{Username: cfg.MongoDBUser, Password: cfg.MongoDBPassword})
	}
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}
	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx2, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}
	return &MongoDB{Client: client, cfg: cfg}, nil
}

// Database returns a database handle. Typically dbName is derived from company_schema.
func (m *MongoDB) Database(dbName string) *mongo.Database {
	if dbName == "" {
		dbName = m.cfg.MongoDBDBName
	}
	return m.Client.Database(dbName)
}

// Collection returns a collection in the given database (e.g. from data_source_name).
func (m *MongoDB) Collection(dbName, collName string) *mongo.Collection {
	return m.Database(dbName).Collection(collName)
}

// Close disconnects the client.
func (m *MongoDB) Close(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}
