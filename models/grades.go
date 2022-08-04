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

const GRADES_PROGRAM_COLLECTION = "grades_programs"
const GRADES_COLLECTION = "grades"

var gradesProgramModel *GradesProgramModel
var gradesModel *GradesModel

type Acumulative struct {
	ID         primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Number     int                `json:"number" bson:"number"`
	Percentage float32            `json:"percentage" bson:"percentage"`
}

type GradesProgram struct {
	ID            primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Module        primitive.ObjectID `json:"module" bson:"module"`
	Number        int                `json:"number" bson:"number"`
	Percentage    float32            `json:"percentage" bson:"percentage"`
	IsAcumulative bool               `json:"is_acumulative" bson:"is_acumulative"`
	Acumulative   []Acumulative      `json:"acumulative,omitempty" bson:"acumulative,omitempty"`
}

type Grade struct {
	ID            primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Module        primitive.ObjectID `json:"module" bson:"module"`
	Student       primitive.ObjectID `json:"student" bson:"student"`
	Program       primitive.ObjectID `json:"program" bson:"program"`
	Acumulative   primitive.ObjectID `json:"acumulative,omitempty" bson:"acumulative,,omitempty"`
	IsAcumulative bool               `json:"is_acumulative" bson:"is_acumulative"`
	Evaluator     primitive.ObjectID `json:"evaluator" bson:"evaluator"`
	Grade         float64            `json:"grade" bson:"grade"`
	Date          primitive.DateTime `json:"date" bson:"date"`
}

type GradeWLookup struct {
	ID          primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Module      primitive.ObjectID `json:"module" bson:"module"`
	Student     primitive.ObjectID `json:"student" bson:"student"`
	Program     primitive.ObjectID `json:"program" bson:"program"`
	Acumulative primitive.ObjectID `json:"acumulative,omitempty" bson:"acumulative,omitempty"`
	Evaluator   SimpleUser         `json:"evaluator" bson:"evaluator"`
	Grade       float64            `json:"grade" bson:"grade"`
	Date        primitive.DateTime `json:"date" bson:"date"`
}

type GradesProgramModel struct {
	CollectionName string
}

type GradesModel struct {
	CollectionName string
}

func NewModelGradesProgram(
	program *forms.GradeProgramForm,
	idModule primitive.ObjectID,
) GradesProgram {
	modelProgram := GradesProgram{
		Module:        idModule,
		Number:        program.Number,
		Percentage:    program.Percentage,
		IsAcumulative: *program.IsAcumulative,
	}
	if *program.IsAcumulative {
		var acumulative []Acumulative
		for _, ac := range program.Acumulative {
			acumulative = append(acumulative, Acumulative{
				ID:         primitive.NewObjectID(),
				Number:     ac.Number,
				Percentage: ac.Percentage,
			})
		}
		modelProgram.Acumulative = acumulative
	}
	return modelProgram
}

func NewModelGrade(
	module,
	student,
	acumulative,
	program,
	evaluator primitive.ObjectID,
	grade float64,
	isAcumulative bool,
) Grade {
	return Grade{
		Module:        module,
		Student:       student,
		Program:       program,
		Evaluator:     evaluator,
		Grade:         grade,
		IsAcumulative: isAcumulative,
		Acumulative:   acumulative,
		Date:          primitive.NewDateTimeFromTime(time.Now()),
	}
}

func (gpModel *GradesProgramModel) Use() *mongo.Collection {
	return DbConnect.GetCollection(gpModel.CollectionName)
}

func (gpModel *GradesProgramModel) GetByID(id primitive.ObjectID) *mongo.SingleResult {
	cursor := gpModel.Use().FindOne(db.Ctx, bson.D{
		{
			Key:   "_id",
			Value: id,
		},
	})
	return cursor
}

func (gpModel *GradesProgramModel) GetOne(filter bson.D) *mongo.SingleResult {
	cursor := gpModel.Use().FindOne(db.Ctx, filter)
	return cursor
}

func (gpModel *GradesProgramModel) GetAll(filter bson.D, options *options.FindOptions) (*mongo.Cursor, error) {
	cursor, err := gpModel.Use().Find(db.Ctx, filter, options)
	return cursor, err
}

func (gpModel *GradesProgramModel) Aggreagate(pipeline mongo.Pipeline) (*mongo.Cursor, error) {
	cursor, err := gpModel.Use().Aggregate(db.Ctx, pipeline)
	return cursor, err
}

func (gpModel *GradesProgramModel) NewDocument(data interface{}) (*mongo.InsertOneResult, error) {
	result, err := gpModel.Use().InsertOne(db.Ctx, data)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (g *GradesModel) Use() *mongo.Collection {
	return DbConnect.GetCollection(g.CollectionName)
}

func (g *GradesModel) GetByID(id primitive.ObjectID) *mongo.SingleResult {
	cursor := g.Use().FindOne(db.Ctx, bson.D{
		{
			Key:   "_id",
			Value: id,
		},
	})
	return cursor
}

func (g *GradesModel) GetOne(filter bson.D) *mongo.SingleResult {
	cursor := g.Use().FindOne(db.Ctx, filter)
	return cursor
}

func (g *GradesModel) GetAll(filter bson.D, options *options.FindOptions) (*mongo.Cursor, error) {
	cursor, err := g.Use().Find(db.Ctx, filter, options)
	return cursor, err
}

func (g *GradesModel) Aggreagate(pipeline mongo.Pipeline) (*mongo.Cursor, error) {
	cursor, err := g.Use().Aggregate(db.Ctx, pipeline)
	return cursor, err
}

func (g *GradesModel) NewDocument(data interface{}) (*mongo.InsertOneResult, error) {
	result, err := g.Use().InsertOne(db.Ctx, data)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func initProgram(collections []string) error {
	for _, collection := range collections {
		if collection == GRADES_PROGRAM_COLLECTION {
			return nil
		}
	}
	var jsonSchema = bson.M{
		"bsonType": "object",
		"required": []string{
			"module",
			"number",
			"percentage",
			"is_acumulative",
		},
		"properties": bson.M{
			"module":      bson.M{"bsonType": "objectId"},
			"number":      bson.M{"bsonType": "int"},
			"percentage":  bson.M{"bsonType": "double"},
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
	err := DbConnect.CreateCollection(GRADES_PROGRAM_COLLECTION, opts)
	if err != nil {
		return err
	}
	return nil
}

func initGrades(collections []string) error {
	for _, collection := range collections {
		if collection == GRADES_COLLECTION {
			return nil
		}
	}
	var jsonSchema = bson.M{
		"bsonType": "object",
		"required": []string{
			"module",
			"student",
			"program",
			"evaluator",
			"is_acumulative",
			"grade",
			"date",
		},
		"properties": bson.M{
			"module":         bson.M{"bsonType": "objectId"},
			"student":        bson.M{"bsonType": "objectId"},
			"program":        bson.M{"bsonType": "objectId"},
			"acumulative":    bson.M{"bsonType": "objectId"},
			"evaluator":      bson.M{"bsonType": "objectId"},
			"grade":          bson.M{"bsonType": "double"},
			"is_acumulative": bson.M{"bsonType": "bool"},
			"date":           bson.M{"bsonType": "date"},
		},
	}
	var validators = bson.M{
		"$jsonSchema": jsonSchema,
	}
	opts := &options.CreateCollectionOptions{
		Validator: validators,
	}
	err := DbConnect.CreateCollection(GRADES_COLLECTION, opts)
	if err != nil {
		return err
	}
	return nil
}

func init() {
	collections, err := DbConnect.GetCollections()
	if err != nil {
		panic(err)
	}
	err = initProgram(collections)
	if err != nil {
		panic(err)
	}
	err = initGrades(collections)
	if err != nil {
		panic(err)
	}
}

func NewGradesProgramModel() Collection {
	if gradesProgramModel == nil {
		gradesProgramModel = &GradesProgramModel{
			CollectionName: GRADES_PROGRAM_COLLECTION,
		}
	}
	return gradesProgramModel
}

func NewGradesModel() Collection {
	if gradesModel == nil {
		gradesModel = &GradesModel{
			CollectionName: GRADES_COLLECTION,
		}
	}
	return gradesModel
}
