package models

import "go.mongodb.org/mongo-driver/bson/primitive"

const SEMESTER_COLLECTION = "semesters"

type Semester struct {
	ID       primitive.ObjectID `json:"_id" bson:"_id" example:"637d5de216f58bc8ec7f7f51"`
	Year     int32              `json:"year" bson:"year" example:"2022"`
	Semester int32              `json:"semester" bson:"semester" example:"2"`
}
