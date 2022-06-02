package res

type Response struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"body"`
}

type Notify struct {
	Title string
	Link  string
	Img   string
	Type  string
}
