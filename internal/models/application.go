package models

import "time"

type Application struct {
	ID        int       `json:"id"`
	JobID     int       `json:"job_id"`
	UserID    int       `json:"user_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
