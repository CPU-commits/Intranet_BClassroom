package forms

import (
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WorkPatternFiles struct {
	Title       string `json:"title" binding:"required,min=1,max=100" validate:"required" minimum:"1" maximum:"100" example:"Title!"`
	Description string `json:"description" binding:"required,min=1,max=300" validate:"required" minimum:"1" maximum:"300" example:"This is a description..."`
	Points      int    `json:"points" binding:"required,min=1" validate:"required" minimum:"1" example:"25"`
}

type WorkPatternWIDFiles struct {
	ID          string `json:"_id" example:"637d5de216f58bc8ec7f7f51"`
	Title       string `json:"title" binding:"required,min=1,max=100" validate:"required" minimum:"1" maximum:"100" example:"Title!"`
	Description string `json:"description" binding:"required,min=1,max=300" validate:"required" minimum:"1" maximum:"300" example:"This is a description..."`
	Points      int    `json:"points" binding:"required,min=1" validate:"required" minimum:"1" example:"25"`
}

type WorkSession struct {
	Block string   `json:"block" example:"637d5de216f58bc8ec7f7f51" binding:"required" validate:"required"`
	Dates []string `json:"dates" binding:"required,dive"`
}

// @Desc grade required if is_qualified==true.
// @Desc pattern required if type == files.
// @Desc time_access in seconds.
// @Desc form_access required if type == form
// @Desc time_access required if form_access = wtime
type WorkForm struct {
	Title          string             `json:"title" binding:"required,min=1,max=100" validate:"required" minimum:"1" maximum:"100" example:"Title!"`
	Description    string             `json:"description" binding:"max=150" maximum:"150" example:"This is a description..."`
	IsQualified    *bool              `json:"is_qualified" binding:"required" validate:"required"`
	Grade          string             `json:"grade,omitempty" binding:"required_if=IsQualified true" example:"637d5de216f58bc8ec7f7f51"`
	Sessions       []WorkSession      `json:"sessions" binding:"required_if=Type in-person,dive"`
	Virtual        *bool              `json:"virtual" binding:"required" validate:"required"`
	Type           string             `json:"type" binding:"required,workType" validate:"required" example:"files" enums:"files,form"`
	Form           string             `json:"form,omitempty" binding:"required_if=Type form" example:"637d5de216f58bc8ec7f7f51"`
	Pattern        []WorkPatternFiles `json:"pattern,omitempty" binding:"required_if=Type files,dive"`
	DateStart      string             `json:"date_start" binding:"required" example:"2006-01-02 15:04"`
	DateLimit      string             `json:"date_limit" binding:"required" example:"2006-01-02 15:04"`
	FormAccess     string             `json:"form_access,omitempty" binding:"required_if=Type form,formAccessType" enums:"default,wtime" example:"wtime"`
	TimeFormAccess int                `json:"time_access,omitempty" binding:"required_if=FormAccess wtime" example:"3600"` // Seconds
	Attached       []Attached         `json:"attached" binding:"omitempty,dive"`
	Acumulative    primitive.ObjectID
}

// @Desc time_access in seconds
type UpdateWorkForm struct {
	Title          string                `json:"title" binding:"min=1,max=100" minimum:"1" maximum:"100" example:"Title!"`
	Description    string                `json:"description" binding:"max=150" maximum:"150" example:"This is a description..."`
	Grade          string                `json:"grade" example:"637d5de216f58bc8ec7f7f51"`
	Form           string                `json:"form" example:"637d5de216f58bc8ec7f7f51"`
	Pattern        []WorkPatternWIDFiles `json:"pattern" binding:"dive"`
	DateStart      string                `json:"date_start" example:"2006-01-02 15:04"`
	DateLimit      string                `json:"date_limit" example:"2006-01-02 15:04"`
	Sessions       []WorkSession         `json:"sessions" binding:"omitempty,dive"`
	FormAccess     string                `json:"form_access,omitempty" binding:"formAccessTypeUp" enums:"default,wtime"`
	TimeFormAccess int                   `json:"time_access,omitempty" example:"3600"` // Seconds
	Attached       []Attached            `json:"attached" binding:"omitempty,dive"`
}

var WorkType validator.Func = func(fl validator.FieldLevel) bool {
	if fl.Field().Interface() == "files" {
		return true
	}
	if fl.Field().Interface() == "form" {
		return true
	}
	if fl.Field().Interface() == "in-person" {
		return true
	}
	return false
}

var FormAccessType validator.Func = func(fl validator.FieldLevel) bool {
	parent := fl.Parent().Interface().(WorkForm)
	if parent.Type == "files" || parent.Type == "in-person" {
		return true
	}
	if fl.Field().Interface() == "default" {
		return true
	}
	if fl.Field().Interface() == "wtime" {
		return true
	}
	return false
}

var FormAccessTypeUpdate validator.Func = func(fl validator.FieldLevel) bool {
	if fl.Field().Interface() == "" {
		return true
	}
	if fl.Field().Interface() == "default" {
		return true
	}
	if fl.Field().Interface() == "wtime" {
		return true
	}
	return false
}
