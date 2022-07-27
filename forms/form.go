package forms

import (
	"github.com/go-playground/validator/v10"
)

type QuestionForm struct {
	ID       string   `json:"_id"`
	Question string   `json:"question" binding:"required,min=3"`
	Type     string   `json:"type" binding:"required,questionType"`
	Answers  []string `json:"answers" binding:"dive,min=3,max=100"`
	Correct  *int     `json:"correct" binding:"required_if=Type alternatives_correct"`
	Points   int      `json:"points" binding:"numeric"`
}

type ItemForm struct {
	ID        string         `json:"_id"`
	Title     string         `json:"title" binding:"required,min=3,max=100"`
	Type      string         `json:"points_type" binding:"itemType"`
	Points    int            `json:"points" binding:"required_if=Type equal"`
	Questions []QuestionForm `json:"questions" binding:"required,dive"`
}

type FormForm struct {
	Title      string     `json:"title" binding:"required,min=3,max=100"`
	PointsType string     `json:"has_points" binding:"required,boolean"`
	Items      []ItemForm `json:"items" binding:"required,omitempty,min=1,dive"`
}

var QuestionType validator.Func = func(fl validator.FieldLevel) bool {
	if fl.Field().Interface() == "alternatives" {
		return true
	}
	if fl.Field().Interface() == "alternatives_correct" {
		return true
	}
	if fl.Field().Interface() == "written" {
		return true
	}
	return false
}

var ItemType validator.Func = func(fl validator.FieldLevel) bool {
	if fl.Field().Interface() == "" {
		return true
	}
	if fl.Field().Interface() == "without" {
		return true
	}
	if fl.Field().Interface() == "equal" {
		return true
	}
	if fl.Field().Interface() == "custom" {
		return true
	}
	return false
}
