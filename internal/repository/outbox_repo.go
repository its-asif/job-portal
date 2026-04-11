package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/its-asif/job-portal/internal/models"
)

type OutboxRepository struct {
	DB *sql.DB
}

func NewOutboxRepository(db *sql.DB) *OutboxRepository {
	return &OutboxRepository{DB: db}
}

func (r *OutboxRepository) CreateEvent(eventType, queueName string, payload []byte) error {
	if r == nil || r.DB == nil {
		return errors.New("database is not configured")
	}

	_, err := r.DB.Exec(`
		INSERT INTO outbox_events (event_type, queue_name, payload, status, attempts, next_attempt_at)
		VALUES ($1, $2, $3, 'pending', 0, NOW())
	`, eventType, queueName, payload)
	return err
}

func (r *OutboxRepository) ListPending(limit int) ([]models.OutboxEvent, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("database is not configured")
	}

	if limit <= 0 {
		limit = 50
	}

	rows, err := r.DB.Query(`
		SELECT id, event_type, queue_name, payload, status, attempts, next_attempt_at, COALESCE(last_error, ''), created_at, updated_at
		FROM outbox_events
		WHERE status = 'pending' AND next_attempt_at <= NOW()
		ORDER BY id ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]models.OutboxEvent, 0)
	for rows.Next() {
		var ev models.OutboxEvent
		if err := rows.Scan(
			&ev.ID,
			&ev.EventType,
			&ev.QueueName,
			&ev.Payload,
			&ev.Status,
			&ev.Attempts,
			&ev.NextAttemptAt,
			&ev.LastError,
			&ev.CreatedAt,
			&ev.UpdatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, ev)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func (r *OutboxRepository) MarkPublished(id int) error {
	if r == nil || r.DB == nil {
		return errors.New("database is not configured")
	}

	_, err := r.DB.Exec(`
		UPDATE outbox_events
		SET status = 'published', updated_at = NOW()
		WHERE id = $1
	`, id)
	return err
}

func (r *OutboxRepository) MarkRetry(id int, lastError string, retryAfter time.Duration) error {
	if r == nil || r.DB == nil {
		return errors.New("database is not configured")
	}

	seconds := int(retryAfter.Seconds())
	if seconds < 1 {
		seconds = 1
	}

	_, err := r.DB.Exec(`
		UPDATE outbox_events
		SET attempts = attempts + 1,
			last_error = $2,
			next_attempt_at = NOW() + ($3 * INTERVAL '1 second'),
			updated_at = NOW()
		WHERE id = $1
	`, id, lastError, seconds)
	return err
}
