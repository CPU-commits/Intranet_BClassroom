package models

import (
	"time"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/elastic/go-elasticsearch/v7/esutil"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const PUBLICATIONS_COLLECTION = "publications"
const PUBLICATIONS_INDEX = "publications"

var publicationModel *PublicationModel

// MongoDB Struct
type Attached struct {
	ID    primitive.ObjectID `json:"_id" bson:"_id,omitempty" example:"637d5de216f58bc8ec7f7f51"`
	Type  string             `json:"type" bson:"type" example:"link" enums:"link,file"` // Link or file
	File  primitive.ObjectID `json:"file,omitempty" bson:"file,omitempty" example:"637d5de216f58bc8ec7f7f51" extensions:"x-omitempty"`
	Link  string             `json:"link,omitempty" bson:"link,omitempty" example:"https://example.com" extensions:"x-omitempty"`
	Title string             `json:"title,omitempty" bson:"title,omitempty" example:"This is a title" extensions:"x-omitempty"`
}

// Sub attached
type AttachedFile struct {
	ID   primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Type string             `json:"type" bson:"type"` // File
	File primitive.ObjectID `json:"file" bson:"file"`
}

type AttachedLink struct {
	ID    primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Type  string             `json:"type" bson:"type"` // Link
	Link  string             `json:"link" bson:"link"`
	Title string             `json:"title" bson:"title"`
}

type Publication struct {
	ID         primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Author     primitive.ObjectID `json:"author" bson:"author"`
	Attached   []Attached         `json:"attached" bson:"attached"`
	SubSection primitive.ObjectID `json:"sub_section" bson:"sub_section"`
	UploadDate primitive.DateTime `json:"upload_date" bson:"upload_date"`
	UpdateDate primitive.DateTime `json:"update_date" bson:"update_date"`
}

// ElasticSearch Struct - Publication content
type ContentPublication struct {
	Content   string    `json:"content"`
	Author    string    `json:"author"`
	IDModule  string    `json:"id_module"`
	Published time.Time `json:"published"`
}

type PublicationModel struct {
	CollectionName string
}

func init() {
	// MongoDB
	collections, err := DbConnect.GetCollections()
	if err != nil {
		panic(err)
	}
	for _, collection := range collections {
		if collection == PUBLICATIONS_COLLECTION {
			return
		}
	}
	var jsonSchema = bson.M{
		"bsonType": "object",
		"required": []string{
			"author",
			"sub_section",
			"upload_date",
			"update_date",
		},
		"properties": bson.M{
			"author":      bson.M{"bsonType": "objectId"},
			"upload_date": bson.M{"bsonType": "date"},
			"update_date": bson.M{"bsonType": "date"},
			"sub_section": bson.M{"bsonType": "objectId"},
			"attached": bson.M{
				"bsonType": bson.A{"array"},
				"items": bson.M{
					"bsonType": "object",
					"required": bson.A{"type"},
					"properties": bson.M{
						"type": bson.M{"enum": bson.A{"link", "file"}},
						"file": bson.M{"bsonType": "objectId"},
						"link": bson.M{"bsonType": "string"},
						"title": bson.M{
							"bsonType":  "string",
							"maxLength": 100,
						},
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
	err = DbConnect.CreateCollection(PUBLICATIONS_COLLECTION, opts)
	if err != nil {
		panic(err)
	}
}

// ElastichSearch Bulk
func NewBulkPublication() (esutil.BulkIndexer, error) {
	es, err := db.NewConnectionEs()
	if err != nil {
		return nil, err
	}

	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:         PUBLICATIONS_INDEX,
		Client:        es,
		NumWorkers:    db.NUM_WORKERS,
		FlushBytes:    int(db.FLUSH_BYTES),
		FlushInterval: db.FLUSH_INTERVAL,
	})
	if err != nil {
		return nil, err
	}
	return bi, nil
}

func NewModelPublication(
	publication *forms.PublicationForm,
	author,
	sectionId primitive.ObjectID,
) (*Publication, []string, error) {
	now := primitive.NewDateTimeFromTime(time.Now())
	attached := []Attached{}
	var attachedIds []string
	for _, attachedData := range publication.Attached {
		attachedId := primitive.NewObjectID()
		attachedIds = append(attachedIds, attachedId.Hex())
		attachedModel := Attached{
			ID:   attachedId,
			Type: attachedData.Type,
		}
		if attachedData.Type == "link" {
			attachedModel.Link = attachedData.Link
			attachedModel.Title = attachedData.Title
		} else {
			fileObjectId, err := primitive.ObjectIDFromHex(attachedData.File)
			if err != nil {
				return nil, nil, err
			}
			attachedModel.File = fileObjectId
		}
		attached = append(attached, attachedModel)
	}
	return &Publication{
		Author:     author,
		Attached:   attached,
		SubSection: sectionId,
		UploadDate: now,
		UpdateDate: now,
	}, attachedIds, nil
}

func (publication *PublicationModel) Use() *mongo.Collection {
	return DbConnect.GetCollection(publication.CollectionName)
}

func (publication *PublicationModel) GetByID(id primitive.ObjectID) *mongo.SingleResult {
	cursor := publication.Use().FindOne(db.Ctx, bson.D{
		{
			Key:   "_id",
			Value: id,
		},
	})
	return cursor
}

func (publication *PublicationModel) GetOne(filter bson.D) *mongo.SingleResult {
	cursor := publication.Use().FindOne(db.Ctx, filter)
	return cursor
}

func (publication *PublicationModel) GetAll(filter bson.D, options *options.FindOptions) (*mongo.Cursor, error) {
	cursor, err := publication.Use().Find(db.Ctx, filter, options)
	return cursor, err
}

func (publication *PublicationModel) Aggreagate(pipeline mongo.Pipeline) (*mongo.Cursor, error) {
	cursor, err := publication.Use().Aggregate(db.Ctx, pipeline)
	return cursor, err
}

func (publication *PublicationModel) NewDocument(data interface{}) (*mongo.InsertOneResult, error) {
	result, err := publication.Use().InsertOne(db.Ctx, data)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func NewPublicationModel() Collection {
	if publicationModel == nil {
		publicationModel = &PublicationModel{
			CollectionName: PUBLICATIONS_COLLECTION,
		}
	}
	return publicationModel
}
