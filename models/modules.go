package models

import (
	"github.com/CPU-commits/Intranet_BClassroom/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const MODULES_COLLECTION = "moduleclasses"

var moduleModel *ModuleModel

type SubSection struct {
	ID   primitive.ObjectID `json:"_id" bson:"_id,omitempty" example:"637d5de216f58bc8ec7f7f51"`
	Name string             `json:"name" bson:"name" example:"Math - Geo"`
}

type Module struct {
	ID          primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Section     primitive.ObjectID `json:"section" bson:"section"`
	Subject     primitive.ObjectID `json:"subject" bson:"subject"`
	Semester    primitive.ObjectID `json:"semester" bson:"semester"`
	Status      bool               `json:"status" bson:"status"` // Finished
	SubSections []SubSection       `json:"sub_sections" bson:"sub_sections"`
	V           int32              `json:"__v" bson:"__v"`
}

type ModuleWithLookup struct {
	ID          primitive.ObjectID `json:"_id" bson:"_id" example:"637d5de216f58bc8ec7f7f51"`
	Section     Section            `json:"section" bson:"section"`
	Subject     Subject            `json:"subject" bson:"subject"`
	Semester    Semester           `json:"semester" bson:"semester"`
	Status      bool               `json:"status" bson:"status"`
	SubSections []SubSection       `json:"sub_sections" bson:"sub_sections"`
	V           int32              `json:"__v" bson:"__v"`
	// Works
	Works []Work `json:"works,omitempty" extensions:"x-omitempty"`
	// If parent - Student Assigned
	Students []*SimpleUser `json:"students,omitempty"`
}

type ModuleModel struct {
	CollectionName string
}

func (module *ModuleModel) Use() *mongo.Collection {
	return DbConnect.GetCollection(module.CollectionName)
}

func (module *ModuleModel) GetByID(id primitive.ObjectID) *mongo.SingleResult {
	cursor := module.Use().FindOne(db.Ctx, bson.D{
		{
			Key:   "_id",
			Value: id,
		},
	})
	return cursor
}

func (module *ModuleModel) GetOne(filter bson.D) *mongo.SingleResult {
	cursor := module.Use().FindOne(db.Ctx, filter)
	return cursor
}

func (module *ModuleModel) GetAll(filter bson.D, options *options.FindOptions) (*mongo.Cursor, error) {
	cursor, err := module.Use().Find(db.Ctx, filter, options)
	return cursor, err
}

func (module *ModuleModel) Aggreagate(pipeline mongo.Pipeline) (*mongo.Cursor, error) {
	cursor, err := module.Use().Aggregate(db.Ctx, pipeline)
	return cursor, err
}

func (module *ModuleModel) NewDocument(data interface{}) (*mongo.InsertOneResult, error) {
	result, err := module.Use().InsertOne(db.Ctx, data)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func NewModuleModel() Collection {
	if moduleModel == nil {
		moduleModel = &ModuleModel{
			CollectionName: MODULES_COLLECTION,
		}
	}
	return moduleModel
}
