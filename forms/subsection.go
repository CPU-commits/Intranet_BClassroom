package forms

type SubSectionData struct {
	SubSection string `json:"sub_section" binding:"min=3,max=100" minimum:"3" maximum:"100" example:"Sub-Section Name"`
}
