package models

import "go.mongodb.org/mongo-driver/bson/primitive"

const SECTION_COLLECTION = "courseletters"

type Section struct {
	ID      primitive.ObjectID `json:"_id" bson:"_id" example:"637d5de216f58bc8ec7f7f51"`
	Section string             `json:"section" bson:"section" example:"A"`
	Course  Course             `json:"course" bson:"course"`
	File    File               `json:"file" bson:"file"`
	V       int32              `json:"__v" bson:"__v"`
}
