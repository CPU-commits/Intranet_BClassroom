package models

import (
	"github.com/CPU-commits/Intranet_BClassroom/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const STUDENTS_COLLECTION = "students"

var studentModel *StudentModel

type Student struct {
	ID                 primitive.ObjectID `json:"_id" bson:"_id"`
	Course             primitive.ObjectID `json:"course" bson:"course"`
	User               primitive.ObjectID `json:"user" bson:"user"`
	RegistrationNumber string             `json:"registation_number" bson:"registration_number"`
	V                  int32              `bson:"__v"`
}

type StudentModel struct {
	CollectionName string
}

func (student *StudentModel) Use() *mongo.Collection {
	return DbConnect.GetCollection(student.CollectionName)
}

func (student *StudentModel) GetByID(id primitive.ObjectID) *mongo.SingleResult {
	cursor := student.Use().FindOne(db.Ctx, bson.D{
		{
			Key:   "_id",
			Value: id,
		},
	})
	return cursor
}

func (student *StudentModel) GetOne(filter bson.D) *mongo.SingleResult {
	cursor := student.Use().FindOne(db.Ctx, filter)
	return cursor
}

func (student *StudentModel) GetAll(filter bson.D, options *options.FindOptions) (*mongo.Cursor, error) {
	cursor, err := student.Use().Find(db.Ctx, filter, options)
	return cursor, err
}

func (student *StudentModel) Aggreagate(pipeline mongo.Pipeline) (*mongo.Cursor, error) {
	cursor, err := student.Use().Aggregate(db.Ctx, pipeline)
	return cursor, err
}

func (student *StudentModel) NewDocument(data interface{}) (*mongo.InsertOneResult, error) {
	result, err := student.Use().InsertOne(db.Ctx, data)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func NewStudentModel() Collection {
	if studentModel == nil {
		studentModel = &StudentModel{
			CollectionName: STUDENTS_COLLECTION,
		}
	}
	return studentModel
}
