package forms

type EvaluateQuestion struct {
	Points *int `json:"points" binding:"required"`
}

type EvaluateFilesForm struct {
	Pattern string `json:"pattern" binding:"required"`
	Points  *int   `json:"points" binding:"required,min=0"`
}
