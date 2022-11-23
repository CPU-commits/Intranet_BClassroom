package models

import "go.mongodb.org/mongo-driver/bson/primitive"

const COURSE_COLLECTION = "courses"

type Course struct {
	ID     primitive.ObjectID `json:"_id" bson:"_id" example:"637d5de216f58bc8ec7f7f51"`
	Course string             `json:"course" bson:"course" example:"First"`
	V      int32              `json:"__v" bson:"__v"`
}
