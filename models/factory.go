package models

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Collection interface {
	Use() *mongo.Collection
	GetByID(id primitive.ObjectID) *mongo.SingleResult
	GetOne(filter bson.D) *mongo.SingleResult
	GetAll(filter bson.D, options *options.FindOptions) (*mongo.Cursor, error)
	Aggreagate(pipeline mongo.Pipeline) (*mongo.Cursor, error)
	NewDocument(data interface{}) (*mongo.InsertOneResult, error)
}
