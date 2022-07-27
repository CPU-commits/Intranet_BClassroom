package models

import "go.mongodb.org/mongo-driver/bson/primitive"

const COURSE_COLLECTION = "courses"

type Course struct {
	ID     primitive.ObjectID `json:"_id" bson:"_id"`
	Course string             `json:"course" bson:"course"`
	V      int32              `json:"__v" bson:"__v"`
}
