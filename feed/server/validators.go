package server

import (
	"github.com/CPU-commits/Intranet_BClassroom/forms"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func InitValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("itemType", forms.ItemType)
		v.RegisterValidation("questionType", forms.QuestionType)
		v.RegisterValidation("workType", forms.WorkType)
		v.RegisterValidation("formAccessType", forms.FormAccessType)
	}
}
