package models

import (
	"github.com/CPU-commits/Intranet_BClassroom/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const MODULES_HISTORY_COLLECTION = "modulehistories"

var moduleHistoryModel *ModuleHistoryModel

type ModuleHistory struct {
	ID       primitive.ObjectID   `bson:"_id"`
	Students []primitive.ObjectID `bson:"students"`
	Module   primitive.ObjectID   `bson:"module"`
	Date     primitive.DateTime   `bson:"date"`
	V        int                  `bson:"__v"`
}

type ModuleHistoryModel struct {
	CollectionName string
}

func (module *ModuleHistoryModel) Use() *mongo.Collection {
	return DbConnect.GetCollection(module.CollectionName)
}

func (module *ModuleHistoryModel) GetByID(id primitive.ObjectID) *mongo.SingleResult {
	cursor := module.Use().FindOne(db.Ctx, bson.D{
		{
			Key:   "_id",
			Value: id,
		},
	})
	return cursor
}

func (module *ModuleHistoryModel) GetOne(filter bson.D) *mongo.SingleResult {
	cursor := module.Use().FindOne(db.Ctx, filter)
	return cursor
}

func (module *ModuleHistoryModel) GetAll(filter bson.D, options *options.FindOptions) (*mongo.Cursor, error) {
	cursor, err := module.Use().Find(db.Ctx, filter, options)
	return cursor, err
}

func (module *ModuleHistoryModel) Aggreagate(pipeline mongo.Pipeline) (*mongo.Cursor, error) {
	cursor, err := module.Use().Aggregate(db.Ctx, pipeline)
	return cursor, err
}

func (module *ModuleHistoryModel) NewDocument(data interface{}) (*mongo.InsertOneResult, error) {
	result, err := module.Use().InsertOne(db.Ctx, data)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func NewModuleHistoryModel() Collection {
	if moduleHistoryModel == nil {
		moduleHistoryModel = &ModuleHistoryModel{
			CollectionName: MODULES_HISTORY_COLLECTION,
		}
	}
	return moduleHistoryModel
}
