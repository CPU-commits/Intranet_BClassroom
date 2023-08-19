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

const SESSIONS_COLLECTION = "sessions"

type Session struct {
	ID       primitive.ObjectID `json:"_id" bson:"_id,omitempty"`
	Student  primitive.ObjectID `json:"student" bson:"student"`
	Work     primitive.ObjectID `json:"work" bson:"work"`
	Block    primitive.ObjectID `json:"block" bson:"block"`
	InDate   primitive.DateTime `json:"in_date" bson:"in_date"`
	PreGrade float64            `json:"pregrade" bson:"pregrade"`
	File     primitive.ObjectID `json:"file,omitempty" bson:"file,omitempty"`
	Date     primitive.DateTime `json:"date" bson:"date"`
}

type SessionWLookup struct {
	ID       primitive.ObjectID      `json:"_id" bson:"_id,omitempty"`
	Student  primitive.ObjectID      `json:"student" bson:"student"`
	Work     primitive.ObjectID      `json:"work" bson:"work"`
	Block    RegisteredCalendarBlock `json:"block" bson:"block"`
	InDate   primitive.DateTime      `json:"in_date" bson:"in_date"`
	PreGrade float64                 `json:"pregrade" bson:"pregrade"`
	File     primitive.ObjectID      `json:"file,omitempty" bson:"file,omitempty"`
	Date     primitive.DateTime      `json:"date" bson:"date"`
}

var sessionModel *SessionModel

type SessionModel struct {
	CollectionName string
}

func (session *SessionModel) Use() *mongo.Collection {
	return DbConnect.GetCollection(session.CollectionName)
}

func (session *SessionModel) GetByID(id primitive.ObjectID) *mongo.SingleResult {
	cursor := session.Use().FindOne(db.Ctx, bson.D{
		{
			Key:   "_id",
			Value: id,
		},
	})
	return cursor
}

func (session *SessionModel) GetOne(filter bson.D) *mongo.SingleResult {
	cursor := session.Use().FindOne(db.Ctx, filter)
	return cursor
}

func (session *SessionModel) GetAll(filter bson.D, options *options.FindOptions) (*mongo.Cursor, error) {
	cursor, err := session.Use().Find(db.Ctx, filter, options)
	return cursor, err
}

func (session *SessionModel) Aggreagate(pipeline mongo.Pipeline) (*mongo.Cursor, error) {
	cursor, err := session.Use().Aggregate(db.Ctx, pipeline)
	return cursor, err
}

func (session *SessionModel) NewDocument(data interface{}) (*mongo.InsertOneResult, error) {
	result, err := session.Use().InsertOne(db.Ctx, data)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func NewModelSession(
	session *forms.EvaluateInperson,
	idStudent,
	idWork primitive.ObjectID,
) (*Session, error) {
	timeDate, err := time.Parse("2006-01-02", session.InDate)
	if err != nil {
		return nil, err
	}
	idObjBlock, err := primitive.ObjectIDFromHex(session.Block)
	if err != nil {
		return nil, err
	}

	return &Session{
		InDate:   primitive.NewDateTimeFromTime(timeDate),
		Student:  idStudent,
		Work:     idWork,
		PreGrade: float64(session.Pregrade),
		Block:    idObjBlock,
		Date:     primitive.NewDateTimeFromTime(time.Now()),
	}, nil
}

func init() {
	collections, err := DbConnect.GetCollections()
	if err != nil {
		panic(err)
	}
	for _, collection := range collections {
		if collection == SESSIONS_COLLECTION {
			return
		}
	}
	var jsonSchema = bson.M{
		"bsonType": "object",
		"required": []string{
			"student",
			"work",
			"block",
			"in_date",
			"pregrade",
			"date",
		},
		"properties": bson.M{
			"student":    bson.M{"bsonType": "objectId"},
			"work":       bson.M{"bsonType": "objectId"},
			"block":      bson.M{"bsonType": "objectId"},
			"file":       bson.M{"bsonType": "objectId"},
			"in_date":    bson.M{"bsonType": "date"},
			"date_limit": bson.M{"bsonType": "date"},
			"pregrade":   bson.M{"bsonType": "double"},
		},
	}
	var validators = bson.M{
		"$jsonSchema": jsonSchema,
	}
	opts := &options.CreateCollectionOptions{
		Validator: validators,
	}
	err = DbConnect.CreateCollection(SESSIONS_COLLECTION, opts)
	if err != nil {
		panic(err)
	}
}

func NewSessionModel() Collection {
	if sessionModel == nil {
		sessionModel = &SessionModel{
			CollectionName: SESSIONS_COLLECTION,
		}
	}
	return sessionModel
}
