package models

import "go.mongodb.org/mongo-driver/bson/primitive"

const SUBJECT_COLLECTION = "subjects"

type Subject struct {
	ID      primitive.ObjectID `json:"_id" bson:"_id"`
	Subject string             `json:"subject" bson:"subject"`
	V       int32              `json:"__v" bson:"__v"`
}
