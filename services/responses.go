package services

import (
	"time"

	"github.com/CPU-commits/Intranet_BClassroom/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ModuleIDs struct {
	IDCourse  primitive.ObjectID
	IDSubject primitive.ObjectID
}

type Acumulative struct {
	ID        string             `json:"_id" example:"63785424db1efbc237faecca"`
	Grade     float64            `json:"grade" example:"70"`
	Evaluator *models.SimpleUser `json:"evaluator"`
	Date      time.Time          `json:"date"`
}

type OrderedGrade struct {
	ID            string            `json:"_id,omitempty" example:"63785424db1efbc237faecca"`
	Grade         float64           `json:"grade" example:"70"`
	IsAcumulative bool              `json:"is_acumulative"`
	Acumulative   []*Acumulative    `json:"acumulative,omitempty" extensions:"x-omitempty"`
	Evaluator     models.SimpleUser `json:"evaluator,omitempty" extensions:"x-omitempty"`
	Date          time.Time         `json:"date,omitempty" extensions:"x-omitempty"`
}

type StudentGrade struct {
	Student models.SimpleUser `json:"student"`
	Grades  []*OrderedGrade   `json:"grades"`
}

type AttachedRes struct {
	ID    string       `json:"_id" example:"637d5de216f58bc8ec7f7f51"`
	Type  string       `json:"type" example:"link" enums:"link,file"`
	File  *models.File `json:"file"`
	Link  string       `json:"link" example:"https://example.com"`
	Title string       `json:"title" example:"This is a title"`
}

type PublicationsRes struct {
	ID         string             `json:"_id" bson:"_id,omitempty" example:"637d5de216f58bc8ec7f7f51"`
	Attached   []AttachedRes      `json:"attached" bson:"attached"`
	Content    interface{}        `json:"content" swaggertype:"string" example:"Content..."`
	UploadDate primitive.DateTime `json:"upload_date" bson:"upload_date" swaggertype:"string" example:"2022-09-21T20:10:23.309+00:00"`
	UpdateDate primitive.DateTime `json:"update_date" bson:"update_date" swaggertype:"string" example:"2022-09-21T20:10:23.309+00:00"`
}

type CloseForm struct {
	Work    string
	Student string
	Diff    float64
}

type Student struct {
	ID                 string                               `json:"_id" example:"637d5de216f58bc8ec7f7f51"`
	User               models.SimpleUser                    `json:"user"`
	V                  int                                  `json:"__v"`
	RegistrationNumber string                               `json:"registration_number" example:"MDVCW45"`
	Course             string                               `json:"course" example:"637d5de216f58bc8ec7f7f51"`
	AccessForm         *models.FormAccess                   `json:"access,omitempty" extensions:"x-omitempty"`
	FilesUploaded      *models.FileUploadedClassroomWLookup `json:"files_uploaded,omitempty" extensions:"x-omitempty"`
	Evuluate           map[string]int                       `json:"evaluate,omitempty" extensions:"x-omitempty"`
}

type AnswerRes struct {
	Answer   models.Answer `json:"answer"`
	Evaluate interface{}   `json:"evaluate,omitempty" extensions:"x-omitempty"`
}

type WorkStatus struct {
	Title       string    `json:"title" example:"This is a title"`
	IsQualified bool      `json:"is_qualified"`
	Type        string    `json:"type" example:"files" enums:"files,form"`
	Module      string    `json:"module" example:"637d5de216f58bc8ec7f7f51"`
	ID          string    `json:"_id" example:"637d5de216f58bc8ec7f7f51"`
	DateStart   time.Time `json:"date_start"`
	DateLimit   time.Time `json:"date_limit"`
	DateUpload  time.Time `json:"date_upload"`
	Status      int       `json:"status"`
}
