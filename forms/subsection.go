package forms

type SubSectionData struct {
	SubSection string `json:"sub_section" binding:"min=3,max=100"`
}
