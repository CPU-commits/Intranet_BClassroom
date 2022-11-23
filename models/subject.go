package models

import "go.mongodb.org/mongo-driver/bson/primitive"

const SUBJECT_COLLECTION = "subjects"

type Subject struct {
	ID      primitive.ObjectID `json:"_id" bson:"_id" example:"637d5de216f58bc8ec7f7f51"`
	Subject string             `json:"subject" bson:"subject" example:"Math"`
	V       int32              `json:"__v" bson:"__v"`
}
