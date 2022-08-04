package forms

// Program
type AcumulativeForm struct {
	Percentage float32 `json:"percentage" binding:"required"`
	Number     int     `json:"number" binding:"required,min=1"`
}

type GradeProgramForm struct {
	Number        int               `json:"number" binding:"required,min=1,max=30"`
	Percentage    float32           `json:"percentage" binding:"required"`
	IsAcumulative *bool             `json:"is_acumulative" binding:"required"`
	Acumulative   []AcumulativeForm `json:"acumulative" binding:"dive"`
}

type GradeForm struct {
	Grade       *float64 `json:"grade" binding:"required"`
	Program     string   `json:"program" binding:"required"`
	Acumulative string   `json:"acumulative"`
}

type UpdateGradeForm struct {
	Grade *float64 `json:"grade" binding:"required"`
}
