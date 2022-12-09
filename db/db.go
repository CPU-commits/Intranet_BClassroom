package db

import (
	"context"
	"fmt"

	"github.com/CPU-commits/Intranet_BClassroom/settings"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const NO_SINGLE_DOCUMENT = "mongo: no documents in result"

var settingsData = settings.GetSettings()
var Ctx = context.TODO()

type MongoClient struct {
	client   *mongo.Client
	database string
}

func newMongoClient(client *mongo.Client, database string) *MongoClient {
	return &MongoClient{
		client:   client,
		database: database,
	}
}

func (mongo *MongoClient) GetCollection(collectionName string) *mongo.Collection {
	collection := mongo.client.Database(mongo.database).Collection(collectionName)
	return collection
}

func (mongo *MongoClient) GetCollections() ([]string, error) {
	filter := bson.D{}
	return mongo.client.Database(mongo.database).ListCollectionNames(Ctx, filter)
}

func (mongo *MongoClient) CreateCollection(collectionName string, opts *options.CreateCollectionOptions) error {
	db := mongo.client.Database(mongo.database)
	return db.CreateCollection(Ctx, collectionName, opts)
}

func NewConnection(host string, dbName string) *MongoClient {
	uri := fmt.Sprintf(
		"%s://%s:%s@%s",
		settingsData.MONGO_CONNECTION,
		settingsData.MONGO_ROOT_USERNAME,
		settingsData.MONGO_ROOT_PASSWORD,
		host,
	)
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(Ctx, clientOptions)
	if err != nil {
		panic(err)
	}
	err = client.Ping(Ctx, nil)
	if err != nil {
		panic(err)
	}
	return newMongoClient(client, dbName)
}
