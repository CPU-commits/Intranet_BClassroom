package forms

// @Desc file required if type == file.
// @Desc link required if type == link.
// @Desc title required if type == link.
type Attached struct {
	Type  string `json:"type" binding:"required" validate:"required" example:"file" enum:"file,link"`
	File  string `json:"file" binding:"required_if=Type file" example:"637d5de216f58bc8ec7f7f51"`
	Link  string `json:"link" binding:"required_if=Type link" example:"https://example.com"`
	Title string `json:"title" binding:"required_if=Type link" example:"Title!"`
}

type PublicationForm struct {
	Content  string     `json:"content" binding:"required,min=3,max=500" validate:"required" minimun:"3" maximum:"500" example:"Content..."`
	Attached []Attached `json:"attached" binding:"omitempty,dive" validate:"optional"`
}

type PublicationUpdateForm struct {
	Content string `json:"content" binding:"required,min=3,max=500" validate:"required" minimum:"3" maximum:"500" example:"Content..."`
}
