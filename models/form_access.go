package models

import (
	"time"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const FORM_ACCESS_COLLECTION = "students_access_form"

var formAcessModel *FormAccessModel

type FormAccess struct {
	ID      primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Student primitive.ObjectID `json:"student" bson:"student"`
	Work    primitive.ObjectID `json:"work" bson:"work"`
	Date    primitive.DateTime `json:"date" bson:"date"`
	Status  string             `json:"status" bson:"status"`
}

type FormAccessModel struct {
	CollectionName string
}

func NewModelFormAccess(studentId, workId primitive.ObjectID) FormAccess {
	return FormAccess{
		Student: studentId,
		Work:    workId,
		Date:    primitive.NewDateTimeFromTime(time.Now()),
		Status:  "opened",
	}
}

func (fa *FormAccessModel) Use() *mongo.Collection {
	return DbConnect.GetCollection(fa.CollectionName)
}

func (fa *FormAccessModel) GetByID(id primitive.ObjectID) *mongo.SingleResult {
	cursor := fa.Use().FindOne(db.Ctx, bson.D{
		{
			Key:   "_id",
			Value: id,
		},
	})
	return cursor
}

func (fa *FormAccessModel) GetOne(filter bson.D) *mongo.SingleResult {
	cursor := fa.Use().FindOne(db.Ctx, filter)
	return cursor
}

func (fa *FormAccessModel) GetAll(filter bson.D, options *options.FindOptions) (*mongo.Cursor, error) {
	cursor, err := fa.Use().Find(db.Ctx, filter, options)
	return cursor, err
}

func (fa *FormAccessModel) Aggreagate(pipeline mongo.Pipeline) (*mongo.Cursor, error) {
	cursor, err := fa.Use().Aggregate(db.Ctx, pipeline)
	return cursor, err
}

func (fa *FormAccessModel) NewDocument(data interface{}) (*mongo.InsertOneResult, error) {
	result, err := fa.Use().InsertOne(db.Ctx, data)
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
		if collection == FORM_ACCESS_COLLECTION {
			return
		}
	}
	var jsonSchema = bson.M{
		"bsonType": "object",
		"required": []string{
			"student",
			"work",
			"date",
			"status",
		},
		"properties": bson.M{
			"student": bson.M{"bsonType": "objectId"},
			"work":    bson.M{"bsonType": "objectId"},
			"date":    bson.M{"bsonType": "date"},
			"status":  bson.M{"enum": bson.A{"opened", "finished", "revised"}},
		},
	}
	var validators = bson.M{
		"$jsonSchema": jsonSchema,
	}
	opts := &options.CreateCollectionOptions{
		Validator: validators,
	}
	err = DbConnect.CreateCollection(FORM_ACCESS_COLLECTION, opts)
	if err != nil {
		panic(err)
	}
}

func NewFormAccessModel() Collection {
	if formAcessModel == nil {
		formAcessModel = &FormAccessModel{
			CollectionName: FORM_ACCESS_COLLECTION,
		}
	}
	return formAcessModel
}
