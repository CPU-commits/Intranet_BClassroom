package models

import (
	"github.com/CPU-commits/Intranet_BClassroom/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const FILES_COLLECTION = "files"

var fileModel *FileModel

type File struct {
	ID          primitive.ObjectID `json:"_id" bson:"_id" example:"637d5de216f58bc8ec7f7f51"`
	Filename    string             `json:"filename" bson:"filename" example:"filename.png"`
	Key         string             `json:"key" bson:"key" example:"$53r34re"`
	Url         string             `json:"url" bson:"url" example:"https://example.com/file"`
	Title       string             `json:"title" bson:"title" example:"Title"`
	Type        string             `json:"type" bson:"type" example:"application/pdf"`
	Status      bool               `json:"status" bson:"status"`
	Permissions string             `json:"permissions" bson:"permissions" enums:"private,public,public_classroom" example:"private"`
	Date        primitive.DateTime `json:"date" bson:"date" swaggertype:"string" example:"2022-09-21T20:10:23.309+00:00"`
}

type Date struct {
	Date int `json:"$date"`
}

type OID struct {
	OID string `json:"$oid"`
}

type FileDB struct {
	ID          OID    `json:"_id"`
	Filename    string `json:"filename"`
	Key         string `json:"key"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	Status      bool   `json:"status"`
	Permissions string `json:"permissions"`
	Date        Date   `json:"date"`
}

type FileModel struct {
	CollectionName string
}

func (file *FileModel) Use() *mongo.Collection {
	return DbConnect.GetCollection(file.CollectionName)
}

func (file *FileModel) GetFileByID(id primitive.ObjectID) (*File, error) {
	var fileData *File

	cursor := file.Use().FindOne(db.Ctx, bson.D{
		{
			Key:   "_id",
			Value: id,
		},
	})
	if err := cursor.Decode(&fileData); err != nil {
		return nil, err
	}
	return fileData, nil
}

func NewFileModel() *FileModel {
	if fileModel == nil {
		fileModel = &FileModel{
			CollectionName: FILES_COLLECTION,
		}
	}
	return fileModel
}
