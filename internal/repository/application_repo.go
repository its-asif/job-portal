package repository

import (
	"database/sql"
	"errors"

	"github.com/its-asif/job-portal/internal/models"
	"github.com/lib/pq"
)

var ErrAlreadyApplied = errors.New("already applied to this job")

type ApplicationRepository struct {
	DB *sql.DB
}

func NewApplicationRepository(db *sql.DB) *ApplicationRepository {
	return &ApplicationRepository{DB: db}
}

func (r *ApplicationRepository) CreateApplication(jobID, userID int) (*models.Application, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("database is not configured")
	}

	query := `
		INSERT INTO applications (job_id, user_id)
		VALUES ($1, $2)
		RETURNING id, job_id, user_id, status, created_at
	`

	var application models.Application
	err := r.DB.QueryRow(query, jobID, userID).Scan(
		&application.ID,
		&application.JobID,
		&application.UserID,
		&application.Status,
		&application.CreatedAt,
	)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return nil, ErrAlreadyApplied
			}
		}
		return nil, err
	}

	return &application, nil
}

func (r *ApplicationRepository) GetApplicationsByJobID(jobID int) ([]models.Application, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("database is not configured")
	}

	query := `
		SELECT id, job_id, user_id, status, created_at
		FROM applications
		WHERE job_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.DB.Query(query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applications := make([]models.Application, 0)
	for rows.Next() {
		var application models.Application
		if err := rows.Scan(
			&application.ID,
			&application.JobID,
			&application.UserID,
			&application.Status,
			&application.CreatedAt,
		); err != nil {
			return nil, err
		}
		applications = append(applications, application)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return applications, nil
}
