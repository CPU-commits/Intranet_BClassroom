package res

const (
	PUBLICATION = "publication"
	WORK        = "work"
	GRADE       = "grade"
)

type Response struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message,omitempty"`
	Data    map[string]interface{} `json:"body,omitempty"`
}

type NotifyClassroom struct {
	Title  string
	Link   string
	Where  string
	Room   string
	Type   string
	IDUser string
}

// Error Response
type ErrorRes struct {
	Err        error
	StatusCode int
}
