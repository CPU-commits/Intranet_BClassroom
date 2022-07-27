package models

import "go.mongodb.org/mongo-driver/bson/primitive"

const SECTION_COLLECTION = "courseletters"

type Section struct {
	ID      primitive.ObjectID `json:"_id" bson:"_id"`
	Section string             `json:"section" bson:"section"`
	Course  Course             `json:"course" bson:"course"`
	File    File               `json:"file" bson:"file"`
	V       int32              `json:"__v" bson:"__v"`
}
