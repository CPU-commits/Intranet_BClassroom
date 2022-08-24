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
	ID        string             `json:"_id"`
	Grade     float64            `json:"grade"`
	Evaluator *models.SimpleUser `json:"evaluator"`
	Date      time.Time          `json:"date"`
}

type OrderedGrade struct {
	ID            string            `json:"_id,omitempty"`
	Grade         float64           `json:"grade"`
	IsAcumulative bool              `json:"is_acumulative"`
	Acumulative   []*Acumulative    `json:"acumulative,omitempty"`
	Evaluator     models.SimpleUser `json:"evaluator,omitempty"`
	Date          time.Time         `json:"date,omitempty"`
}

type StudentGrade struct {
	Student models.SimpleUser `json:"student"`
	Grades  []*OrderedGrade   `json:"grades"`
}

type AttachedRes struct {
	ID    string       `json:"_id"`
	Type  string       `json:"type"`
	File  *models.File `json:"file"`
	Link  string       `json:"link"`
	Title string       `json:"title"`
}

type PublicationsRes struct {
	ID         string             `json:"_id" bson:"_id,omitempty"`
	Attached   []AttachedRes      `json:"attached" bson:"attached"`
	Content    interface{}        `json:"content"`
	UploadDate primitive.DateTime `json:"upload_date" bson:"upload_date"`
	UpdateDate primitive.DateTime `json:"update_date" bson:"update_date"`
}

type CloseForm struct {
	Work    string
	Student string
	Diff    float64
}

type Student struct {
	ID                 string                               `json:"_id"`
	User               models.SimpleUser                    `json:"user"`
	V                  int                                  `json:"__v"`
	RegistrationNumber string                               `json:"registration_number"`
	Course             string                               `json:"course"`
	AccessForm         *models.FormAccess                   `json:"access,omitempty"`
	FilesUploaded      *models.FileUploadedClassroomWLookup `json:"files_uploaded,omitempty"`
	Evuluate           map[string]int                       `json:"evaluate,omitempty"`
}

type AnswerRes struct {
	Answer   models.Answer `json:"answer"`
	Evaluate interface{}   `json:"evaluate,omitempty"`
}
