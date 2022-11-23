package models

import (
	"time"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const ANSWERS_COLLECTION = "answers"

var answerModel *AnswerModel

type Answer struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" example:"637d5de216f58bc8ec7f7f51"`
	Student  primitive.ObjectID `bson:"student" example:"637d5de216f58bc8ec7f7f51"`
	Work     primitive.ObjectID `bson:"work" example:"637d5de216f58bc8ec7f7f51"`
	Question primitive.ObjectID `json:"question" bson:"question" example:"637d5de216f58bc8ec7f7f51"`
	Answer   int                `json:"answer" bson:"answer" example:"1"`
	Response string             `json:"response" bson:"response,omitempty" example:"Response..."`
	Date     primitive.DateTime `json:"date" bson:"date" swaggertype:"string" example:"2022-09-21T20:10:23.309+00:00"`
}

type AnswerModel struct {
	CollectionName string
}

func init() {
	collections, err := DbConnect.GetCollections()
	if err != nil {
		panic(err)
	}
	for _, collection := range collections {
		if collection == ANSWERS_COLLECTION {
			return
		}
	}
	var jsonSchema = bson.M{
		"bsonType": "object",
		"required": []string{
			"student",
			"work",
			"date",
			"question",
		},
		"properties": bson.M{
			"student":  bson.M{"bsonType": "objectId"},
			"work":     bson.M{"bsonType": "objectId"},
			"question": bson.M{"bsonType": "objectId"},
			"date":     bson.M{"bsonType": "date"},
			"answer":   bson.M{"bsonType": "int"},
			"response": bson.M{"bsonType": "string"},
		},
	}
	var validators = bson.M{
		"$jsonSchema": jsonSchema,
	}
	opts := &options.CreateCollectionOptions{
		Validator: validators,
	}
	err = DbConnect.CreateCollection(ANSWERS_COLLECTION, opts)
	if err != nil {
		panic(err)
	}
}

func NewModelAnswer(answer *forms.AnswerForm, student, work, question primitive.ObjectID) *Answer {
	modelAnswer := &Answer{
		Student:  student,
		Work:     work,
		Question: question,
		Date:     primitive.NewDateTimeFromTime(time.Now()),
	}
	if answer.Answer != nil {
		modelAnswer.Answer = *answer.Answer
	} else {
		modelAnswer.Response = answer.Response
	}
	return modelAnswer
}

func (answer *AnswerModel) Use() *mongo.Collection {
	return DbConnect.GetCollection(answer.CollectionName)
}

func (answer *AnswerModel) GetByID(id primitive.ObjectID) *mongo.SingleResult {
	cursor := answer.Use().FindOne(db.Ctx, bson.D{
		{
			Key:   "_id",
			Value: id,
		},
	})
	return cursor
}

func (answer *AnswerModel) GetOne(filter bson.D) *mongo.SingleResult {
	cursor := answer.Use().FindOne(db.Ctx, filter)
	return cursor
}

func (answer *AnswerModel) GetAll(filter bson.D, options *options.FindOptions) (*mongo.Cursor, error) {
	cursor, err := answer.Use().Find(db.Ctx, filter, options)
	return cursor, err
}

func (answer *AnswerModel) Aggreagate(pipeline mongo.Pipeline) (*mongo.Cursor, error) {
	cursor, err := answer.Use().Aggregate(db.Ctx, pipeline)
	return cursor, err
}

func (answer *AnswerModel) NewDocument(data interface{}) (*mongo.InsertOneResult, error) {
	result, err := answer.Use().InsertOne(db.Ctx, data)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func NewAnswerModel() Collection {
	if answerModel == nil {
		answerModel = &AnswerModel{
			CollectionName: ANSWERS_COLLECTION,
		}
	}
	return answerModel
}
