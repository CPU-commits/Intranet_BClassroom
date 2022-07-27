package models

import (
	"time"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const FILE_UPLOADED_CLASSROOM_COLLECTION = "file_uploaded_classroom"

var fileUCModel *FileUploadedClassroomModel

type EvaluatedFiles struct {
	ID      primitive.ObjectID `json:"_id" bson:"_id"`
	Pattern primitive.ObjectID `json:"pattern" bson:"pattern"`
	Points  int                `json:"points" bson:"points"`
}

type FileUploadedClassroom struct {
	ID            primitive.ObjectID   `json:"_id" bson:"_id,omitempty"`
	Work          primitive.ObjectID   `json:"work" bson:"work"`
	Student       primitive.ObjectID   `json:"student" bson:"student"`
	FilesUploaded []primitive.ObjectID `json:"files_uploaded" bson:"files_uploaded"`
	Evaluate      []EvaluatedFiles     `json:"evaluate,omitempty" bson:"evaluate,omitempty"`
	Date          primitive.DateTime   `json:"date" bson:"date"`
}

type FileUploadedClassroomWLookup struct {
	ID            primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Work          primitive.ObjectID `json:"work" bson:"work"`
	Student       primitive.ObjectID `json:"student" bson:"student"`
	FilesUploaded []File             `json:"files_uploaded" bson:"files_uploaded"`
	Evaluate      []EvaluatedFiles   `json:"evaluate,omitempty" bson:"evaluate,omitempty"`
	Date          primitive.DateTime `json:"date" bson:"date"`
}

type FileUploadedClassroomModel struct {
	CollectionName string
}

func NewModelFileUC(idWork, idUser primitive.ObjectID, files []primitive.ObjectID) FileUploadedClassroom {
	return FileUploadedClassroom{
		Work:          idWork,
		Student:       idUser,
		FilesUploaded: files,
		Date:          primitive.NewDateTimeFromTime(time.Now()),
	}
}

func (fUC *FileUploadedClassroomModel) Use() *mongo.Collection {
	return DbConnect.GetCollection(fUC.CollectionName)
}

func (fUC *FileUploadedClassroomModel) GetByID(id primitive.ObjectID) *mongo.SingleResult {
	cursor := fUC.Use().FindOne(db.Ctx, bson.D{
		{
			Key:   "_id",
			Value: id,
		},
	})
	return cursor
}

func (fUC *FileUploadedClassroomModel) GetOne(filter bson.D) *mongo.SingleResult {
	cursor := fUC.Use().FindOne(db.Ctx, filter)
	return cursor
}

func (fUC *FileUploadedClassroomModel) GetAll(filter bson.D, options *options.FindOptions) (*mongo.Cursor, error) {
	cursor, err := fUC.Use().Find(db.Ctx, filter, options)
	return cursor, err
}

func (fUC *FileUploadedClassroomModel) Aggreagate(pipeline mongo.Pipeline) (*mongo.Cursor, error) {
	cursor, err := fUC.Use().Aggregate(db.Ctx, pipeline)
	return cursor, err
}

func (fUC *FileUploadedClassroomModel) NewDocument(data interface{}) (*mongo.InsertOneResult, error) {
	result, err := fUC.Use().InsertOne(db.Ctx, data)
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
		if collection == FILE_UPLOADED_CLASSROOM_COLLECTION {
			return
		}
	}
	var jsonSchema = bson.M{
		"bsonType": "object",
		"required": []string{
			"work",
			"student",
			"date",
			"files_uploaded",
		},
		"properties": bson.M{
			"work":    bson.M{"bsonType": "objectId"},
			"student": bson.M{"bsonType": "objectId"},
			"date":    bson.M{"bsonType": "date"},
			"attached": bson.M{
				"bsonType": bson.A{"array"},
				"items": bson.M{
					"bsonType": "objectId",
				},
			},
			"evaluate": bson.M{
				"bsonType": bson.A{"array"},
				"items": bson.M{
					"bsonType": "object",
					"required": bson.A{
						"_id",
						"points",
						"pattern",
					},
					"properties": bson.M{
						"_id":     bson.M{"bsonType": "objectId"},
						"points":  bson.M{"bsonType": "int"},
						"pattern": bson.M{"bsonType": "objectId"},
					},
				},
			},
		},
	}
	var validators = bson.M{
		"$jsonSchema": jsonSchema,
	}
	opts := &options.CreateCollectionOptions{
		Validator: validators,
	}
	err = DbConnect.CreateCollection(FILE_UPLOADED_CLASSROOM_COLLECTION, opts)
	if err != nil {
		panic(err)
	}
}

func NewFileUCModel() Collection {
	if fileUCModel == nil {
		fileUCModel = &FileUploadedClassroomModel{
			CollectionName: FILE_UPLOADED_CLASSROOM_COLLECTION,
		}
	}
	return fileUCModel
}
