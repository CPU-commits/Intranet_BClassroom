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

const FORM_COLLECTION = "forms"
const FORM_QUESTION_COLLECTION = "form_questions"

var formModel *FormModel
var formQuestionModel *FormQuestionModel

// Types
type ItemQuestion struct {
	ID       primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Type     string             `json:"type" bson:"type"`
	Question string             `json:"question" bson:"question"`
	Answers  []string           `json:"answers" bson:"answers,omitempty"`
	Points   int                `json:"points,omitempty" bson:"points,omitempty"`
	Correct  int                `json:"correct" bson:"correct,omitempty"`
}

type FormItem struct {
	ID         primitive.ObjectID   `json:"_id" bson:"_id,omitempty"`
	Title      string               `json:"title" bson:"title"`
	PointsType string               `json:"points_type,omiempty" bson:"points_type,omiempty"`
	Questions  []primitive.ObjectID `json:"questions" bson:"questions"`
}

type Form struct {
	ID         primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Author     primitive.ObjectID `json:"author" bson:"author"`
	Title      string             `json:"title" bson:"title"`
	HasPoints  bool               `json:"has_points" bson:"has_points"`
	Items      []FormItem         `json:"items" bson:"items"`
	UploadDate primitive.DateTime `json:"upload_date" bson:"upload_date"`
	UpdateDate primitive.DateTime `json:"update_date" bson:"update_date"`
	Status     bool               `bson:"status"`
}

// Types With Lookup
type FormItemWLookup struct {
	ID         primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Title      string             `json:"title" bson:"title"`
	PointsType string             `json:"points_type,omiempty" bson:"points_type,omiempty"`
	Questions  []ItemQuestion     `json:"questions" bson:"questions"`
}

type FormWLookup struct {
	ID         primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Title      string             `json:"title" bson:"title"`
	HasPoints  bool               `json:"has_points" bson:"has_points"`
	Items      []FormItemWLookup  `json:"items" bson:"items"`
	UploadDate primitive.DateTime `json:"upload_date" bson:"upload_date"`
	UpdateDate primitive.DateTime `json:"update_date" bson:"update_date"`
}

type FormModel struct {
	CollectionName string
}
type FormQuestionModel struct {
	CollectionName string
}

func initForms(collections []string) {
	for _, collection := range collections {
		if collection == FORM_COLLECTION {
			return
		}
	}
	var jsonSchema = bson.M{
		"bsonType": "object",
		"required": []string{
			"title",
			"has_points",
			"items",
			"author",
			"upload_date",
			"update_date",
		},
		"properties": bson.M{
			"title":       bson.M{"bsonType": "string", "maxLength": 100},
			"author":      bson.M{"bsonType": "objectId"},
			"upload_date": bson.M{"bsonType": "date"},
			"update_date": bson.M{"bsonType": "date"},
			"has_points":  bson.M{"bsonType": "bool"},
			"status":      bson.M{"bsonType": "bool"},
			"items": bson.M{
				"bsonType": bson.A{"array"},
				"items": bson.M{
					"bsonType": "object",
					"required": bson.A{"title", "questions"},
					"properties": bson.M{
						"points_type": bson.M{"enum": bson.A{"equal", "custom", "without"}},
						"questions": bson.M{
							"bsonType": bson.A{"array"},
							"items": bson.M{
								"bsonType": "objectId",
							},
						},
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
	err := DbConnect.CreateCollection(FORM_COLLECTION, opts)
	if err != nil {
		panic(err)
	}
}

func initQuestions(collections []string) {
	for _, collection := range collections {
		if collection == FORM_QUESTION_COLLECTION {
			return
		}
	}
	var jsonSchema = bson.M{
		"bsonType": "object",
		"required": []string{
			"type",
			"question",
		},
		"properties": bson.M{
			"type": bson.M{
				"enum": bson.A{
					"alternatives",
					"alternatives_correct",
					"written",
				},
			},
			"question": bson.M{"bsonType": "string"},
			"answers": bson.M{
				"bsonType": bson.A{"array"},
				"items": bson.M{
					"bsonType": "string",
				},
			},
			"points":  bson.M{"bsonType": "int"},
			"correct": bson.M{"bsonType": "int"},
		},
	}
	var validators = bson.M{
		"$jsonSchema": jsonSchema,
	}
	opts := &options.CreateCollectionOptions{
		Validator: validators,
	}
	err := DbConnect.CreateCollection(FORM_QUESTION_COLLECTION, opts)
	if err != nil {
		panic(err)
	}
}

func init() {
	// MongoDB
	collections, err := DbConnect.GetCollections()
	if err != nil {
		panic(err)
	}
	initForms(collections)
	initQuestions(collections)
}

// Form
func NewModelsForm(
	form *forms.FormForm,
	questions [][]primitive.ObjectID,
	formType string,
	author primitive.ObjectID,
) *Form {
	var items []FormItem
	for i, item := range form.Items {
		itemData := FormItem{
			ID:        primitive.NewObjectID(),
			Title:     item.Title,
			Questions: questions[i],
		}
		if formType == "true" {
			itemData.PointsType = item.Type
		} else {
			itemData.PointsType = "without"
		}
		items = append(items, itemData)
	}

	now := primitive.NewDateTimeFromTime(time.Now())
	return &Form{
		Title:      form.Title,
		HasPoints:  form.PointsType == "true",
		Items:      items,
		Author:     author,
		UploadDate: now,
		UpdateDate: now,
		Status:     true,
	}
}

func (form *FormModel) Use() *mongo.Collection {
	return DbConnect.GetCollection(form.CollectionName)
}

func (form *FormModel) GetByID(id primitive.ObjectID) *mongo.SingleResult {
	cursor := form.Use().FindOne(db.Ctx, bson.D{
		{
			Key:   "_id",
			Value: id,
		},
	})
	return cursor
}

func (form *FormModel) GetOne(filter bson.D) *mongo.SingleResult {
	cursor := form.Use().FindOne(db.Ctx, filter)
	return cursor
}

func (form *FormModel) GetAll(filter bson.D, options *options.FindOptions) (*mongo.Cursor, error) {
	cursor, err := form.Use().Find(db.Ctx, filter, options)
	return cursor, err
}

func (form *FormModel) Aggreagate(pipeline mongo.Pipeline) (*mongo.Cursor, error) {
	cursor, err := form.Use().Aggregate(db.Ctx, pipeline)
	return cursor, err
}

func (form *FormModel) NewDocument(data interface{}) (*mongo.InsertOneResult, error) {
	result, err := form.Use().InsertOne(db.Ctx, data)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Form Question model
func NewModelsFormQuestion(
	question *forms.QuestionForm,
	item *forms.ItemForm,
	points int,
) *ItemQuestion {
	questionData := ItemQuestion{
		Type:     question.Type,
		Question: question.Question,
	}
	if question.Type != "alternatives" {
		questionData.Points = points
	}
	if question.Type == "alternatives_correct" {
		questionData.Correct = *question.Correct
	}
	questionData.Answers = question.Answers

	return &questionData
}

func (formQuestion *FormQuestionModel) Use() *mongo.Collection {
	return DbConnect.GetCollection(formQuestion.CollectionName)
}

func (formQuestion *FormQuestionModel) GetByID(id primitive.ObjectID) *mongo.SingleResult {
	cursor := formQuestion.Use().FindOne(db.Ctx, bson.D{
		{
			Key:   "_id",
			Value: id,
		},
	})
	return cursor
}

func (formQuestion *FormQuestionModel) GetOne(filter bson.D) *mongo.SingleResult {
	cursor := formQuestion.Use().FindOne(db.Ctx, filter)
	return cursor
}

func (formQuestion *FormQuestionModel) GetAll(filter bson.D, options *options.FindOptions) (*mongo.Cursor, error) {
	cursor, err := formQuestion.Use().Find(db.Ctx, filter, options)
	return cursor, err
}

func (formQuestion *FormQuestionModel) Aggreagate(pipeline mongo.Pipeline) (*mongo.Cursor, error) {
	cursor, err := formQuestion.Use().Aggregate(db.Ctx, pipeline)
	return cursor, err
}

func (formQuestion *FormQuestionModel) NewDocument(data interface{}) (*mongo.InsertOneResult, error) {
	result, err := formQuestion.Use().InsertOne(db.Ctx, data)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// New models
func NewFormModel() Collection {
	if formModel == nil {
		formModel = &FormModel{
			CollectionName: FORM_COLLECTION,
		}
	}
	return formModel
}

func NewFormQuestionModel() Collection {
	if formQuestionModel == nil {
		formQuestionModel = &FormQuestionModel{
			CollectionName: FORM_QUESTION_COLLECTION,
		}
	}
	return formQuestionModel
}
