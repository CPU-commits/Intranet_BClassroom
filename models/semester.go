package models

import "go.mongodb.org/mongo-driver/bson/primitive"

const SEMESTER_COLLECTION = "semesters"

type Semester struct {
	ID       primitive.ObjectID `json:"_id" bson:"_id"`
	Year     int32              `json:"year" bson:"year"`
	Semester int32              `json:"semester" bson:"semester"`
}
