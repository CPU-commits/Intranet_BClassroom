package forms

// Program
type AcumulativeForm struct {
	Percentage float32 `json:"percentage" binding:"required" validate:"required" example:"45"`
	Number     int     `json:"number" binding:"required,min=1" validate:"required" minimum:"1" example:"1"`
}

type GradeProgramForm struct {
	Number        int               `json:"number" binding:"required,min=1,max=30" validate:"required" minimum:"1" maximum:"30" example:"1"`
	Percentage    float32           `json:"percentage" binding:"required" validate:"required" example:"25"`
	IsAcumulative *bool             `json:"is_acumulative" binding:"required" validate:"required"`
	Acumulative   []AcumulativeForm `json:"acumulative" binding:"dive" validate:"optional"`
}

type GradeForm struct {
	Grade       *float64 `json:"grade" binding:"required" validate:"required" example:"70"`
	Program     string   `json:"program" binding:"required" validate:"required" example:"637ab12ae976057567a4d67d"`
	Acumulative string   `json:"acumulative" validate:"optional" example:"637ab12ae976057567a4d67d"`
}

type UpdateGradeForm struct {
	Grade *float64 `json:"grade" binding:"required" validate:"required" example:"55"`
}
