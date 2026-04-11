package models

import "time"

type OutboxEvent struct {
	ID            int       `json:"id"`
	EventType     string    `json:"event_type"`
	QueueName     string    `json:"queue_name"`
	Payload       []byte    `json:"payload"`
	Status        string    `json:"status"`
	Attempts      int       `json:"attempts"`
	NextAttemptAt time.Time `json:"next_attempt_at"`
	LastError     string    `json:"last_error,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
