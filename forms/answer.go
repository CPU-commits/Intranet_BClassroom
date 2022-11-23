package forms

type AnswerForm struct {
	Answer   *int   `json:"answer" example:"1" validate:"optional"`
	Response string `json:"response" example:"Response..." validate:"optional"`
}

type Answer struct {
	Question string `json:"question" binding:"required" validate:"required"`
	Answer   *int   `json:"answer" example:"1" validate:"optional"`
	Response string `json:"response" example:"Response..." validate:"optional"`
}

type AnswersForm struct {
	Answers []Answer `json:"answers" binding:"required,dive" validate:"required"`
}
