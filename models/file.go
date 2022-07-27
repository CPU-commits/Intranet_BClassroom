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
	ID          primitive.ObjectID `json:"_id" bson:"_id"`
	Filename    string             `json:"filename" bson:"filename"`
	Key         string             `json:"key" bson:"key"`
	Url         string             `json:"url" bson:"url"`
	Title       string             `json:"title" bson:"title"`
	Type        string             `json:"type" bson:"type"`
	Status      bool               `json:"status" bson:"status"`
	Permissions string             `json:"permissions" bson:"permissions"`
	Date        primitive.DateTime `json:"date" bson:"date"`
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
