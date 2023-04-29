package forms

import (
	"github.com/go-playground/validator/v10"
)

// @Description correct required if type == "alternatives_correct"
// @Description points required if (father)item.type != "equal"
type QuestionForm struct {
	ID       string   `json:"_id" validate:"optional" example:"637846a8bc4ea33de990c098"`
	Question string   `json:"question" binding:"required,min=3" validate:"required" minimum:"3" example:"Who is...?"`
	Type     string   `json:"type" binding:"required,questionType" validate:"required" enums:"alternatives,alternatives_correct,written" example:"alternatives"`
	Answers  []string `json:"answers" binding:"dive,min=1,max=100" minimum:"3" maximum:"100" example:"a, b, c"`
	Correct  *int     `json:"correct" binding:"required_if=Type alternatives_correct" example:"0"`
	Points   int      `json:"points" binding:"numeric" example:"21"`
}

// @Description points required if type == "equal"
type ItemForm struct {
	ID        string         `json:"_id" validate:"required" example:"637846a8bc4ea33de990c098"`
	Title     string         `json:"title" binding:"required,min=3,max=100" validate:"required" minimum:"3" maximum:"100" example:"Item Form Title !"`
	Type      string         `json:"points_type" binding:"itemType" enums:"without,equal,custom" example:"without"`
	Points    int            `json:"points" binding:"required_if=Type equal" validate:"required" example:"25"`
	Questions []QuestionForm `json:"questions" binding:"required,dive" validate:"required"`
}

type FormForm struct {
	Title      string     `json:"title" binding:"required,min=3,max=100" validate:"required" minimum:"3" maximum:"100" example:"Form title"`
	PointsType string     `json:"has_points" binding:"required,boolean" validate:"optional"`
	Items      []ItemForm `json:"items" binding:"required,omitempty,min=1,dive" validate:"required"`
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
