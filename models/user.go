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
	ID             string `json:"_id,omitempty" example:"63785424db1efbc237faecca"`
	Name           string `json:"name,omitempty" bson:"name" example:"Name" extensions:"x-omitempty"`
	FirstLastname  string `json:"first_lastname,omitempty" bson:"first_lastname" example:"FirstLastname" extensions:"x-omitempty"`
	SecondLastname string `json:"second_lastname,omitempty" bson:"second_lastname" example:"SecondLastname" extensions:"x-omitempty"`
	Rut            string `json:"rut,omitempty" bson:"rut" example:"12345678-9" extensions:"x-omitempty"`
}
