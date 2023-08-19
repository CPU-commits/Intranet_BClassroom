package models

import "go.mongodb.org/mongo-driver/bson/primitive"

const CALENDAR_COLLECTION = "calendars"
const CALENDAR_BLOCK_COLLECTION = "calendar_blocks"

type CalendarBlock struct {
	ID         primitive.ObjectID `json:"_id" bson:"_id"`
	HourStart  string             `json:"hour_start" bson:"hour_start"`
	HourFinish string             `json:"hour_finish" bson:"hour_finish"`
	Number     int                `json:"number" bson:"number"`
}

type RegisteredCalendarBlock struct {
	ID      primitive.ObjectID `json:"_id" bson:"_id"`
	Day     string             `json:"day" bson:"day"`
	Block   CalendarBlock      `json:"block" bson:"block"`
	Subject primitive.ObjectID `json:"subject" bson:"subject"`
}
