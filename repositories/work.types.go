package repositories

import "time"

type WorkStatus struct {
	Title       string    `json:"title" example:"This is a title"`
	IsQualified bool      `json:"is_qualified"`
	Type        string    `json:"type" example:"files" enums:"files,form"`
	Module      string    `json:"module" example:"637d5de216f58bc8ec7f7f51"`
	ID          string    `json:"_id" example:"637d5de216f58bc8ec7f7f51"`
	DateStart   time.Time `json:"date_start"`
	DateLimit   time.Time `json:"date_limit"`
	DateUpload  time.Time `json:"date_upload"`
	Status      int       `json:"status"`
}
