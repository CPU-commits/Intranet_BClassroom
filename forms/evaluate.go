package forms

type EvaluateQuestion struct {
	Points *int `json:"points" binding:"required" validate:"required" example:"25"`
}

type EvaluateFilesForm struct {
	Pattern string `json:"pattern" binding:"required" validate:"required" example:"Pattern"`
	Points  *int   `json:"points" binding:"required,min=0" validate:"required" example:"25" minimum:"0"`
}

type EvaluateInperson struct {
	InDate   string  `json:"in_date" binding:"required" validate:"required"`
	Block    string  `json:"block" binding:"required" validate:"required"`
	Pregrade float32 `json:"pregrade" binding:"required" validate:"required" example:"6.5"`
}
