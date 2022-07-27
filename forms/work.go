package forms

import (
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type WorkPatternForm struct {
	Title       string `json:"title" binding:"required,min=1,max=100"`
	Description string `json:"description" binding:"required,min=1,max=300"`
	Points      int    `json:"points" binding:"required,min=1"`
}

type WorkForm struct {
	Title          string            `json:"title" binding:"required,min=1,max=100"`
	Description    string            `json:"description" bson:"max=150"`
	IsQualified    *bool             `json:"is_qualified" binding:"required"`
	Grade          string            `json:"grade,omitempty" binding:"required_if=IsQualified true"`
	Type           string            `json:"type" binding:"required,workType"`
	Form           string            `json:"form,omitempty" binding:"required_if=Type form"`
	Pattern        []WorkPatternForm `json:"pattern,omitempty" binding:"required_if=Type files,dive"`
	DateStart      string            `json:"date_start" binding:"required"`
	DateLimit      string            `json:"date_limit" binding:"required"`
	FormAccess     string            `json:"form_access,omitempty" binding:"required_if=Type form,formAccessType"`
	TimeFormAccess int               `json:"time_access,omitempty" binding:"required_if=FormAccess wtime"` // Seconds
	Attached       []Attached        `json:"attached" binding:"omitempty,dive"`
	Acumulative    primitive.ObjectID
}

var WorkType validator.Func = func(fl validator.FieldLevel) bool {
	if fl.Field().Interface() == "files" {
		return true
	}
	if fl.Field().Interface() == "form" {
		return true
	}
	return false
}

var FormAccessType validator.Func = func(fl validator.FieldLevel) bool {
	parent := fl.Parent().Interface().(WorkForm)
	if parent.Type == "files" {
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
