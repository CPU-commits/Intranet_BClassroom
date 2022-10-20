package models

import (
	"github.com/CPU-commits/Intranet_BClassroom/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const AVERAGES_COLLECTION = "averages"

var averageModel *AverageModel

type Average struct {
	ID       primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Average  float64            `json:"average" bson:"average"`
	Semester primitive.ObjectID `json:"semester,omitempty" bson:"semester"`
	Student  primitive.ObjectID `json:"student" bson:"student"`
}

type AverageModel struct {
	CollectionName string
}

func NewModelAverage(average float64, semester, student primitive.ObjectID) Average {
	return Average{
		Average:  average,
		Semester: semester,
		Student:  student,
	}
}

func (g *AverageModel) Use() *mongo.Collection {
	return DbConnect.GetCollection(g.CollectionName)
}

func (g *AverageModel) GetByID(id primitive.ObjectID) *mongo.SingleResult {
	cursor := g.Use().FindOne(db.Ctx, bson.D{
		{
			Key:   "_id",
			Value: id,
		},
	})
	return cursor
}

func (g *AverageModel) GetOne(filter bson.D) *mongo.SingleResult {
	cursor := g.Use().FindOne(db.Ctx, filter)
	return cursor
}

func (g *AverageModel) GetAll(filter bson.D, options *options.FindOptions) (*mongo.Cursor, error) {
	cursor, err := g.Use().Find(db.Ctx, filter, options)
	return cursor, err
}

func (g *AverageModel) Aggreagate(pipeline mongo.Pipeline) (*mongo.Cursor, error) {
	cursor, err := g.Use().Aggregate(db.Ctx, pipeline)
	return cursor, err
}

func (g *AverageModel) NewDocument(data interface{}) (*mongo.InsertOneResult, error) {
	result, err := g.Use().InsertOne(db.Ctx, data)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func init() {
	collections, err := DbConnect.GetCollections()
	if err != nil {
		panic(err)
	}
	for _, collection := range collections {
		if collection == AVERAGES_COLLECTION {
			return
		}
	}
	var jsonSchema = bson.M{
		"bsonType": "object",
		"required": []string{
			"student",
			"average",
			"semester",
		},
		"properties": bson.M{
			"semester": bson.M{"bsonType": "objectId"},
			"student":  bson.M{"bsonType": "objectId"},
			"average":  bson.M{"bsonType": "double"},
		},
	}
	var validators = bson.M{
		"$jsonSchema": jsonSchema,
	}
	opts := &options.CreateCollectionOptions{
		Validator: validators,
	}
	err = DbConnect.CreateCollection(AVERAGES_COLLECTION, opts)
	if err != nil {
		panic(err)
	}
}

func NewAveragesModel() Collection {
	if averageModel == nil {
		averageModel = &AverageModel{
			CollectionName: AVERAGES_COLLECTION,
		}
	}
	return averageModel
}
