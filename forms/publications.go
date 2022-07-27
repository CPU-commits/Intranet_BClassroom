package forms

type Attached struct {
	Type  string `json:"type" binding:"required"`
	File  string `json:"file" binding:"required_if=Type file"`
	Link  string `json:"link" binding:"required_if=Type link"`
	Title string `json:"title" binding:"required_if=Type link"`
}

type PublicationForm struct {
	Content  string     `json:"content" binding:"required,min=3,max=500"`
	Attached []Attached `json:"attached" binding:"omitempty,dive"`
}

type PublicationUpdateForm struct {
	Content string `json:"content" binding:"required,min=3,max=500"`
}
