package models

import "time"

type Job struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Location    string    `json:"location"`
	Salary      int64     `json:"salary"`
	Company     string    `json:"company"`
	PostedBy    int       `json:"posted_by"`
	CreatedAt   time.Time `json:"created_at"`
}
