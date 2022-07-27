package forms

type AnswerForm struct {
	Answer   *int   `json:"answer"`
	Response string `json:"response"`
}

type Answer struct {
	Question string `json:"question" binding:"required"`
	Answer   *int   `json:"answer"`
	Response string `json:"response"`
}

type AnswersForm struct {
	Answers []Answer `json:"answers" binding:"required,dive"`
}
