package models

import (
	"time"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const WORK_GRADES_COLLECTION = "work_grades"

var workGradesModel *WorkGradesModel

type WorkGrade struct {
	ID        primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Module    primitive.ObjectID `json:"module" bson:"module"`
	Student   primitive.ObjectID `json:"student" bson:"student"`
	Work      primitive.ObjectID `json:"work" bson:"work"`
	Evaluator primitive.ObjectID `json:"evaluator" bson:"evaluator"`
	Grade     float64            `json:"grade" bson:"grade"`
	Date      primitive.DateTime `json:"date" bson:"date"`
}

type WorkGradeWLookup struct {
	ID        primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Module    primitive.ObjectID `json:"module" bson:"module"`
	Work      primitive.ObjectID `json:"work" bson:"work"`
	Student   primitive.ObjectID `json:"student" bson:"student"`
	Evaluator SimpleUser         `json:"evaluator" bson:"evaluator"`
	Grade     float64            `json:"grade" bson:"grade"`
	Date      primitive.DateTime `json:"date" bson:"date"`
}

type WorkGradesModel struct {
	CollectionName string
}

func NewModelWorkGrade(
	module,
	student,
	evaluator primitive.ObjectID,
	grade float64,
) WorkGrade {
	return WorkGrade{
		Module:    module,
		Student:   student,
		Evaluator: evaluator,
		Grade:     grade,
		Date:      primitive.NewDateTimeFromTime(time.Now()),
	}
}

func (g *WorkGradesModel) Use() *mongo.Collection {
	return DbConnect.GetCollection(g.CollectionName)
}

func (g *WorkGradesModel) GetByID(id primitive.ObjectID) *mongo.SingleResult {
	cursor := g.Use().FindOne(db.Ctx, bson.D{
		{
			Key:   "_id",
			Value: id,
		},
	})
	return cursor
}

func (g *WorkGradesModel) GetOne(filter bson.D) *mongo.SingleResult {
	cursor := g.Use().FindOne(db.Ctx, filter)
	return cursor
}

func (g *WorkGradesModel) GetAll(filter bson.D, options *options.FindOptions) (*mongo.Cursor, error) {
	cursor, err := g.Use().Find(db.Ctx, filter, options)
	return cursor, err
}

func (g *WorkGradesModel) Aggreagate(pipeline mongo.Pipeline) (*mongo.Cursor, error) {
	cursor, err := g.Use().Aggregate(db.Ctx, pipeline)
	return cursor, err
}

func (g *WorkGradesModel) NewDocument(data interface{}) (*mongo.InsertOneResult, error) {
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
		if collection == WORK_GRADES_COLLECTION {
			return
		}
	}
	var jsonSchema = bson.M{
		"bsonType": "object",
		"required": []string{
			"student",
			"work",
			"module",
			"date",
			"evaluator",
		},
		"properties": bson.M{
			"student":   bson.M{"bsonType": "objectId"},
			"work":      bson.M{"bsonType": "objectId"},
			"module":    bson.M{"bsonType": "objectId"},
			"evaluator": bson.M{"bsonType": "objectId"},
			"date":      bson.M{"bsonType": "date"},
		},
	}
	var validators = bson.M{
		"$jsonSchema": jsonSchema,
	}
	opts := &options.CreateCollectionOptions{
		Validator: validators,
	}
	err = DbConnect.CreateCollection(WORK_GRADES_COLLECTION, opts)
	if err != nil {
		panic(err)
	}
}

func NewWorkGradesModel() Collection {
	if workGradesModel == nil {
		workGradesModel = &WorkGradesModel{
			CollectionName: WORK_GRADES_COLLECTION,
		}
	}
	return workGradesModel
}
