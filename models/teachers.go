package models

import (
	"github.com/CPU-commits/Intranet_BClassroom/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const TEACHERS_COLLECTION = "teachers"

var teacherModel *TeacherModel

type TeacherImparted struct {
	ID      primitive.ObjectID `json:"_id" bson:"_id"`
	Subject primitive.ObjectID `json:"subject" bson:"subject"`
	Course  primitive.ObjectID `json:"course" bson:"course"`
}

type Teacher struct {
	ID       primitive.ObjectID `json:"_id" bson:"_id"`
	User     primitive.ObjectID `json:"user" bson:"user"`
	Imparted []TeacherImparted  `json:"imparted" bson:"imparted"`
	V        int32              `bson:"__v"`
}

type TeacherModel struct {
	CollectionName string
}

func (teacher *TeacherModel) Use() *mongo.Collection {
	return DbConnect.GetCollection(teacher.CollectionName)
}

func (teacher *TeacherModel) GetByID(id primitive.ObjectID) *mongo.SingleResult {
	cursor := teacher.Use().FindOne(db.Ctx, bson.D{
		{
			Key:   "_id",
			Value: id,
		},
	})
	return cursor
}

func (teacher *TeacherModel) GetOne(filter bson.D) *mongo.SingleResult {
	cursor := teacher.Use().FindOne(db.Ctx, filter)
	return cursor
}

func (teacher *TeacherModel) GetAll(filter bson.D, options *options.FindOptions) (*mongo.Cursor, error) {
	cursor, err := teacher.Use().Find(db.Ctx, filter, options)
	return cursor, err
}

func (teacher *TeacherModel) Aggreagate(pipeline mongo.Pipeline) (*mongo.Cursor, error) {
	cursor, err := teacher.Use().Aggregate(db.Ctx, pipeline)
	return cursor, err
}

func (teacher *TeacherModel) NewDocument(data interface{}) (*mongo.InsertOneResult, error) {
	result, err := teacher.Use().InsertOne(db.Ctx, data)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func NewTeacherModel() Collection {
	if teacherModel == nil {
		teacherModel = &TeacherModel{
			CollectionName: TEACHERS_COLLECTION,
		}
	}
	return teacherModel
}
