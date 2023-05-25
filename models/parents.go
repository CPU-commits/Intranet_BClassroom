package models

import (
	"github.com/CPU-commits/Intranet_BClassroom/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const PARENTS_COLLECTION = "parents"

var parentModel *ParentsModel

type Parent struct {
	ID       primitive.ObjectID   `json:"_id" bson:"_id"`
	User     primitive.ObjectID   `json:"user" bson:"user"`
	Students []primitive.ObjectID `json:"students" bson:"students"`
	V        int32                `bson:"__v"`
}

type ParentsModel struct {
	CollectionName string
}

func (parent *ParentsModel) Use() *mongo.Collection {
	return DbConnect.GetCollection(parent.CollectionName)
}

func (parent *ParentsModel) GetByID(id primitive.ObjectID) *mongo.SingleResult {
	cursor := parent.Use().FindOne(db.Ctx, bson.D{
		{
			Key:   "_id",
			Value: id,
		},
	})
	return cursor
}

func (parent *ParentsModel) GetOne(filter bson.D) *mongo.SingleResult {
	cursor := parent.Use().FindOne(db.Ctx, filter)
	return cursor
}

func (parent *ParentsModel) GetAll(filter bson.D, options *options.FindOptions) (*mongo.Cursor, error) {
	cursor, err := parent.Use().Find(db.Ctx, filter, options)
	return cursor, err
}

func (parent *ParentsModel) Aggreagate(pipeline mongo.Pipeline) (*mongo.Cursor, error) {
	cursor, err := parent.Use().Aggregate(db.Ctx, pipeline)
	return cursor, err
}

func (parent *ParentsModel) NewDocument(data interface{}) (*mongo.InsertOneResult, error) {
	result, err := parent.Use().InsertOne(db.Ctx, data)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func NewParentsModel() Collection {
	if parentModel == nil {
		parentModel = &ParentsModel{
			CollectionName: PARENTS_COLLECTION,
		}
	}
	return parentModel
}
