package models

import (
	"time"

	"github.com/CPU-commits/Intranet_BClassroom/db"
	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const WORKS_COLLECTION = "works"
const WORKS_INDEX = "works"

var worksModel *WorkModel

type WorkSession struct {
	Block primitive.ObjectID   `json:"block" bson:"block"`
	Dates []primitive.DateTime `json:"dates" bson:"dates"`
}

type WorkPattern struct {
	ID          primitive.ObjectID `json:"_id" bson:"_id" example:"637d5de216f58bc8ec7f7f51"`
	Title       string             `json:"title" bson:"title" example:"Pattern!"`
	Description string             `json:"description" bson:"description" example:"This is a description"`
	Points      int                `json:"points" bson:"points" example:"25"`
}

// Mongodb
type Work struct {
	ID             primitive.ObjectID `json:"_id" bson:"_id,omitempty" example:"637d5de216f58bc8ec7f7f51"`
	Author         primitive.ObjectID `json:"author" bson:"author" example:"637d5de216f58bc8ec7f7f51"`
	Module         primitive.ObjectID `bson:"module" example:"637d5de216f58bc8ec7f7f51"`
	Title          string             `json:"title" bson:"title" example:"Work!"`
	Description    string             `json:"description,omitempty" bson:"description,omitempty" example:"This is a description" extensions:"x-omitempty"`
	IsQualified    bool               `json:"is_qualified" bson:"is_qualified"`
	Grade          primitive.ObjectID `json:"grade,omitempty" bson:"grade,omitempty" example:"637d5de216f58bc8ec7f7f51" extensions:"x-omitempty"`
	Acumulative    primitive.ObjectID `json:"acumulative" bson:"acumulative,omitempty" example:"637d5de216f58bc8ec7f7f51" extensions:"x-omitempty"`
	Type           string             `json:"type" bson:"type" example:"form" enums:"files,form"`
	Form           primitive.ObjectID `json:"form,omitempty" bson:"form,omitempty" example:"637d5de216f58bc8ec7f7f51" extensions:"x-omitempty"`
	Pattern        []WorkPattern      `json:"pattern,omitempty" bson:"pattern,omitempty" extensions:"x-omitempty"`
	DateStart      primitive.DateTime `json:"date_start" bson:"date_start" swaggertype:"string" example:"2022-09-21T20:10:23.309+00:00"`
	DateLimit      primitive.DateTime `json:"date_limit" bson:"date_limit" swaggertype:"string" example:"2022-09-21T20:10:23.309+00:00"`
	FormAccess     string             `json:"form_access,omitempty" bson:"form_access,omitempty" example:"default" enums:"default,wtime" extensions:"x-omitempty"`
	TimeFormAccess int                `json:"time_access,omitempty" bson:"time_access,omitempty" example:"2" extensions:"x-omitempty"`
	IsRevised      bool               `json:"is_revised" bson:"is_revised"`
	Virtual        bool               `json:"virtual" bson:"virtual"`
	Sessions       []WorkSession      `json:"sessions" bson:"sessions,omitempty"`
	Attached       []Attached         `json:"attached,omitempty" bson:"attached,omitempty" extensions:"x-omitempty"`
	DateUpload     primitive.DateTime `json:"date_upload" bson:"date_upload" swaggertype:"string" example:"2022-09-21T20:10:23.309+00:00"`
	DateUpdate     primitive.DateTime `json:"date_update" bson:"date_update" swaggertype:"string" example:"2022-09-21T20:10:23.309+00:00"`
}

type WorkWLookup struct {
	ID             primitive.ObjectID        `json:"_id" bson:"_id,omitempty" example:"637d5de216f58bc8ec7f7f51"`
	Module         primitive.ObjectID        `bson:"module" example:"637d5de216f58bc8ec7f7f51"`
	Author         SimpleUser                `json:"author" bson:"author"`
	Title          string                    `json:"title" bson:"title" example:"This is a title"`
	Description    string                    `json:"description,omitempty" bson:"description,omitempty" example:"This is a description" extensions:"x-omitempty"`
	Form           primitive.ObjectID        `json:"form,omitempty" bson:"form,omitempty" example:"637d5de216f58bc8ec7f7f51" extensions:"x-omitempty"`
	IsQualified    bool                      `json:"is_qualified" bson:"is_qualified"`
	Grade          GradesProgram             `json:"grade,omitempty" bson:"grade,omitempty" extensions:"x-omitempty"`
	Acumulative    primitive.ObjectID        `json:"acumulative,omitempty" bson:"acumulative,omitempty" example:"637d5de216f58bc8ec7f7f51" extensions:"x-omitempty"`
	Type           string                    `json:"type" bson:"type" example:"files" enums:"files,form"`
	Pattern        []WorkPattern             `json:"pattern,omitempty" bson:"pattern,omitempty" extensions:"x-omitempty"`
	DateStart      primitive.DateTime        `json:"date_start" bson:"date_start" swaggertype:"string" example:"2022-09-21T20:10:23.309+00:00"`
	DateLimit      primitive.DateTime        `json:"date_limit" bson:"date_limit" swaggertype:"string" example:"2022-09-21T20:10:23.309+00:00"`
	IsRevised      bool                      `json:"is_revised" bson:"is_revised"`
	FormAccess     string                    `json:"form_access,omitempty" bson:"form_access,omitempty" example:"default" extensions:"x-omitempty" enums:"default,wtime"`
	Virtual        bool                      `json:"virtual" bson:"virtual"`
	Sessions       []WorkSession             `json:"sessions" bson:"sessions,omitempty"`
	Blocks         []RegisteredCalendarBlock `json:"blocks,omitempty" bson:"blocks,omitempty"`
	TimeFormAccess int                       `json:"time_access,omitempty" bson:"time_access,omitempty" example:"2" extensions:"x-omitempty"`
	Attached       []Attached                `json:"attached,omitempty" bson:"attached,omitempty"`
	DateUpload     primitive.DateTime        `json:"date_upload" bson:"date_upload" swaggertype:"string" example:"2022-09-21T20:10:23.309+00:00"`
	DateUpdate     primitive.DateTime        `json:"date_update" bson:"date_update" swaggertype:"string" example:"2022-09-21T20:10:23.309+00:00"`
}

type AttachedRes struct {
	ID    string `json:"_id" bson:"_id"`
	Type  string `json:"type" bson:"type"`
	File  *File  `json:"file,omitempty" bson:"file"`
	Link  string `json:"link,omitempty" bson:"link"`
	Title string `json:"title,omitempty" bson:"title"`
}

type WorkWLookupNFiles struct {
	ID             primitive.ObjectID        `json:"_id" bson:"_id,omitempty" example:"637d5de216f58bc8ec7f7f51"`
	Module         primitive.ObjectID        `bson:"module" example:"637d5de216f58bc8ec7f7f51"`
	Author         SimpleUser                `json:"author" bson:"author"`
	Title          string                    `json:"title" bson:"title" example:"This is a title"`
	Description    string                    `json:"description,omitempty" bson:"description,omitempty" example:"This is a description" extensions:"x-omitempty"`
	Form           primitive.ObjectID        `json:"form,omitempty" bson:"form,omitempty" example:"637d5de216f58bc8ec7f7f51" extensions:"x-omitempty"`
	IsQualified    bool                      `json:"is_qualified" bson:"is_qualified"`
	Grade          GradesProgram             `json:"grade,omitempty" bson:"grade,omitempty" extensions:"x-omitempty"`
	Acumulative    primitive.ObjectID        `json:"acumulative,omitempty" bson:"acumulative,omitempty" example:"637d5de216f58bc8ec7f7f51" extensions:"x-omitempty"`
	Type           string                    `json:"type" bson:"type" example:"files" enums:"files,form"`
	Pattern        []WorkPattern             `json:"pattern,omitempty" bson:"pattern,omitempty" extensions:"x-omitempty"`
	DateStart      primitive.DateTime        `json:"date_start" bson:"date_start" swaggertype:"string" example:"2022-09-21T20:10:23.309+00:00"`
	DateLimit      primitive.DateTime        `json:"date_limit" bson:"date_limit" swaggertype:"string" example:"2022-09-21T20:10:23.309+00:00"`
	IsRevised      bool                      `json:"is_revised" bson:"is_revised"`
	FormAccess     string                    `json:"form_access,omitempty" bson:"form_access,omitempty" example:"default" extensions:"x-omitempty" enums:"default,wtime"`
	TimeFormAccess int                       `json:"time_access,omitempty" bson:"time_access,omitempty" extensions:"x-omitempty"`
	Virtual        bool                      `json:"virtual" bson:"virtual"`
	Sessions       []WorkSession             `json:"sessions" bson:"sessions,omitempty"`
	Blocks         []RegisteredCalendarBlock `json:"blocks" bson:"blocks,omitempty"`
	Attached       []AttachedRes             `json:"attached,omitempty" bson:"attached,omitempty" extensions:"x-omitempty"`
	DateUpload     primitive.DateTime        `json:"date_upload" bson:"date_upload" swaggertype:"string" example:"2022-09-21T20:10:23.309+00:00"`
	DateUpdate     primitive.DateTime        `json:"date_update" bson:"date_update" swaggertype:"string" example:"2022-09-21T20:10:23.309+00:00"`
}

// ElasticSearch Struct - Work indexer
type WorkES struct {
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	DateStart   time.Time `json:"date_start"`
	DateLimit   time.Time `json:"date_limit"`
	Author      string    `json:"author"`
	IDModule    string    `json:"id_module"`
	Published   time.Time `json:"published"`
}

type WorkModel struct {
	CollectionName string
}

func NewModelWork(
	work *forms.WorkForm,
	dateStart,
	dateLimit time.Time,
	idModule,
	userId primitive.ObjectID,
) (*Work, error) {
	now := primitive.NewDateTimeFromTime(time.Now())

	modelWork := &Work{
		Author:      userId,
		Module:      idModule,
		Description: work.Description,
		Title:       work.Title,
		IsQualified: *work.IsQualified,
		Type:        work.Type,
		DateStart:   primitive.NewDateTimeFromTime(dateStart),
		DateLimit:   primitive.NewDateTimeFromTime(dateLimit),
		IsRevised:   false,
		Virtual:     *work.Virtual,
		DateUpload:  now,
		DateUpdate:  now,
	}
	if *work.IsQualified {
		idObjGrade, err := primitive.ObjectIDFromHex(work.Grade)
		if err != nil {
			return nil, err
		}
		modelWork.Grade = idObjGrade
		if !work.Acumulative.IsZero() {
			modelWork.Acumulative = work.Acumulative
		}
	}
	if work.Type == "form" {
		idObjForm, _ := primitive.ObjectIDFromHex(work.Form)
		modelWork.Form = idObjForm
		modelWork.FormAccess = work.FormAccess
		modelWork.TimeFormAccess = work.TimeFormAccess
	}
	if work.Type == "files" {
		var pattern []WorkPattern
		for _, item := range work.Pattern {
			pattern = append(pattern, WorkPattern{
				ID:          primitive.NewObjectID(),
				Title:       item.Title,
				Description: item.Description,
				Points:      item.Points,
			})
		}

		modelWork.Pattern = pattern
	}
	if work.Type == "in-person" {
		var sessions []WorkSession

		for _, session := range work.Sessions {
			idObjBlock, err := primitive.ObjectIDFromHex(session.Block)
			if err != nil {
				return nil, err
			}
			var dates []primitive.DateTime
			for _, date := range session.Dates {
				time, err := time.Parse("2006-01-02", date)
				if err != nil {
					return nil, err
				}

				dates = append(dates, primitive.NewDateTimeFromTime(time))
			}

			sessions = append(sessions, WorkSession{
				Block: idObjBlock,
				Dates: dates,
			})
		}

		modelWork.Sessions = sessions
	}
	// Attached
	if len(work.Attached) > 0 {
		var attacheds []Attached
		for _, att := range work.Attached {
			attached := Attached{
				ID:   primitive.NewObjectID(),
				Type: att.Type,
			}
			if att.Type == "file" {
				idObjFile, err := primitive.ObjectIDFromHex(att.File)
				if err != nil {
					return nil, err
				}
				attached.File = idObjFile
			} else if attached.Type == "link" {
				attached.Link = att.Link
				attached.Title = att.Title
			}
			attacheds = append(attacheds, attached)
		}
		modelWork.Attached = attacheds
	}
	return modelWork, nil
}

func (work *WorkModel) Use() *mongo.Collection {
	return DbConnect.GetCollection(work.CollectionName)
}

func (work *WorkModel) GetByID(id primitive.ObjectID) *mongo.SingleResult {
	cursor := work.Use().FindOne(db.Ctx, bson.D{
		{
			Key:   "_id",
			Value: id,
		},
	})
	return cursor
}

func (work *WorkModel) GetOne(filter bson.D) *mongo.SingleResult {
	cursor := work.Use().FindOne(db.Ctx, filter)
	return cursor
}

func (work *WorkModel) GetAll(filter bson.D, options *options.FindOptions) (*mongo.Cursor, error) {
	cursor, err := work.Use().Find(db.Ctx, filter, options)
	return cursor, err
}

func (work *WorkModel) Aggreagate(pipeline mongo.Pipeline) (*mongo.Cursor, error) {
	cursor, err := work.Use().Aggregate(db.Ctx, pipeline)
	return cursor, err
}

func (work *WorkModel) NewDocument(data interface{}) (*mongo.InsertOneResult, error) {
	result, err := work.Use().InsertOne(db.Ctx, data)
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
		if collection == WORKS_COLLECTION {
			return
		}
	}
	var jsonSchema = bson.M{
		"bsonType": "object",
		"required": []string{
			"author",
			"title",
			"is_qualified",
			"is_revised",
			"module",
			"type",
			"date_start",
			"date_limit",
			"date_upload",
			"date_update",
			"virtual",
		},
		"properties": bson.M{
			"author":       bson.M{"bsonType": "objectId"},
			"module":       bson.M{"bsonType": "objectId"},
			"title":        bson.M{"bsonType": "string", "maxLength": 100},
			"description":  bson.M{"bsonType": "string", "maxLength": 150},
			"is_qualified": bson.M{"bsonType": "bool"},
			"is_revised":   bson.M{"bsonType": "bool"},
			"grade":        bson.M{"bsonType": "objectId"},
			"acumulative":  bson.M{"bsonType": "objectId"},
			"type":         bson.M{"enum": bson.A{"files", "form", "in-person"}},
			"form":         bson.M{"bsonType": "objectId"},
			"date_start":   bson.M{"bsonType": "date"},
			"date_limit":   bson.M{"bsonType": "date"},
			"date_upload":  bson.M{"bsonType": "date"},
			"date_update":  bson.M{"bsonType": "date"},
			"virtual":      bson.M{"bsonType": "bool"},
			"sessions": bson.M{
				"bsonType": bson.A{"array"},
				"items": bson.M{
					"bsonType": "object",
					"required": bson.A{
						"block",
						"dates",
					},
					"properties": bson.M{
						"block": bson.M{
							"bsonType": "objectId",
						},
						"dates": bson.M{
							"bsonType": bson.A{"array"},
							"items": bson.M{
								"bsonType": "date",
							},
						},
					},
				},
			},
			"form_access": bson.M{"enum": bson.A{"default", "wtime"}},
			"time_access": bson.M{"bsonType": "int", "minimum": 1},
			"pattern": bson.M{
				"bsonType": bson.A{"array"},
				"items": bson.M{
					"bsonType": "object",
					"required": bson.A{
						"title",
						"description",
						"points",
					},
					"properties": bson.M{
						"title": bson.M{
							"bsonType":  "string",
							"maxLength": 100,
						},
						"description": bson.M{
							"bsonType":  "string",
							"maxLength": 300,
						},
						"points": bson.M{
							"bsonType": "int",
							"minimum":  1,
						},
					},
				},
			},
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
	err = DbConnect.CreateCollection(WORKS_COLLECTION, opts)
	if err != nil {
		panic(err)
	}
}

// ElastichSearch Bulk
func NewBulkWork() (esutil.BulkIndexer, error) {
	es, err := db.NewConnectionEs()
	if err != nil {
		return nil, err
	}

	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:         WORKS_INDEX,
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

func NewWorkModel() Collection {
	if worksModel == nil {
		worksModel = &WorkModel{
			CollectionName: WORKS_COLLECTION,
		}
	}
	return worksModel
}
