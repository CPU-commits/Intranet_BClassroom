package models

type UserTypes string

const USERS_COLLECTION = "users"

const (
	DIRECTOR          = "f"
	DIRECTIVE         = "e"
	TEACHER           = "d"
	ATTORNEY          = "c"
	STUDENT_DIRECTIVE = "b"
	STUDENT           = "a"
)

type SimpleUser struct {
	ID             string `json:"_id,omitempty"`
	Name           string `json:"name,omitempty" bson:"name"`
	FirstLastname  string `json:"first_lastname,omitempty" bson:"first_lastname"`
	SecondLastname string `json:"second_lastname,omitempty" bson:"second_lastname"`
	Rut            string `json:"rut,omitempty" bson:"rut"`
}
