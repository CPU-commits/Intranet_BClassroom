package models

import (
	"time"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const EVALUATED_ANSWERS_COLLECTION = "evaluated_answers"

var evaluatedAnswersModel *EvaluatedAnswersModel

type EvaluatedAnswers struct {
	ID        primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Question  primitive.ObjectID `json:"question" bson:"question"`
	Student   primitive.ObjectID `json:"student" bson:"student"`
	Work      primitive.ObjectID `json:"work" bson:"work"`
	Evaluator primitive.ObjectID `json:"evaluator" bson:"evaluator"`
	Points    int                `json:"points" bson:"points"`
	Date      primitive.DateTime `json:"date" bson:"date"`
}

type EvaluatedAnswersWLookup struct {
	ID        primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Evaluator SimpleUser         `json:"evaluator" bson:"evaluator"`
	Points    int                `json:"points" bson:"points"`
	Date      primitive.DateTime `json:"date" bson:"date"`
}

type EvaluatedAnswersModel struct {
	CollectionName string
}

func NewModelEvaluatedAnswers(
	points int,
	idStudent,
	idQuestion,
	idWork,
	idEvaluator primitive.ObjectID,
) EvaluatedAnswers {
	return EvaluatedAnswers{
		Question:  idQuestion,
		Student:   idStudent,
		Work:      idWork,
		Evaluator: idEvaluator,
		Points:    points,
		Date:      primitive.NewDateTimeFromTime(time.Now()),
	}
}

func init() {
	// MongoDB
	collections, err := DbConnect.GetCollections()
	if err != nil {
		panic(err)
	}
	for _, collection := range collections {
		if collection == EVALUATED_ANSWERS_COLLECTION {
			return
		}
	}
	var jsonSchema = bson.M{
		"bsonType": "object",
		"required": []string{
			"question",
			"student",
			"work",
			"evaluator",
			"points",
			"date",
		},
		"properties": bson.M{
			"question":  bson.M{"bsonType": "objectId"},
			"student":   bson.M{"bsonType": "objectId"},
			"work":      bson.M{"bsonType": "objectId"},
			"evaluator": bson.M{"bsonType": "objectId"},
			"date":      bson.M{"bsonType": "date"},
			"points":    bson.M{"bsonType": "int"},
		},
	}
	var validators = bson.M{
		"$jsonSchema": jsonSchema,
	}
	opts := &options.CreateCollectionOptions{
		Validator: validators,
	}
	err = DbConnect.CreateCollection(EVALUATED_ANSWERS_COLLECTION, opts)
	if err != nil {
		panic(err)
	}
}

func (eA *EvaluatedAnswersModel) Use() *mongo.Collection {
	return DbConnect.GetCollection(eA.CollectionName)
}

func (eA *EvaluatedAnswersModel) GetByID(id primitive.ObjectID) *mongo.SingleResult {
	cursor := eA.Use().FindOne(db.Ctx, bson.D{
		{
			Key:   "_id",
			Value: id,
		},
	})
	return cursor
}

func (eA *EvaluatedAnswersModel) GetOne(filter bson.D) *mongo.SingleResult {
	cursor := eA.Use().FindOne(db.Ctx, filter)
	return cursor
}

func (eA *EvaluatedAnswersModel) GetAll(filter bson.D, options *options.FindOptions) (*mongo.Cursor, error) {
	cursor, err := eA.Use().Find(db.Ctx, filter, options)
	return cursor, err
}

func (eA *EvaluatedAnswersModel) Aggreagate(pipeline mongo.Pipeline) (*mongo.Cursor, error) {
	cursor, err := eA.Use().Aggregate(db.Ctx, pipeline)
	return cursor, err
}

func (eA *EvaluatedAnswersModel) NewDocument(data interface{}) (*mongo.InsertOneResult, error) {
	result, err := eA.Use().InsertOne(db.Ctx, data)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func NewEvaluatedAnswersModel() Collection {
	if evaluatedAnswersModel == nil {
		evaluatedAnswersModel = &EvaluatedAnswersModel{
			CollectionName: EVALUATED_ANSWERS_COLLECTION,
		}
	}
	return evaluatedAnswersModel
}
