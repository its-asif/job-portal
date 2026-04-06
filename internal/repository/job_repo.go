package repository

import (
	"database/sql"
	"errors"

	"github.com/its-asif/job-portal/internal/models"
)

var ErrJobNotFound = errors.New("job not found")

type JobRepository struct {
	DB *sql.DB
}

type UpdateJobInput struct {
	Title       *string
	Description *string
	Location    *string
	Salary      *int64
	Company     *string
}

func NewJobRepository(db *sql.DB) *JobRepository {
	return &JobRepository{DB: db}
}

func (r *JobRepository) CreateJob(job *models.Job) error {
	if r == nil || r.DB == nil {
		return errors.New("database is not configured")
	}

	query := `
		INSERT INTO jobs (title, description, location, salary, company, posted_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	return r.DB.QueryRow(
		query,
		job.Title,
		job.Description,
		job.Location,
		job.Salary,
		job.Company,
		job.PostedBy,
	).Scan(&job.ID, &job.CreatedAt)
}

func (r *JobRepository) GetJobByID(id int) (*models.Job, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("database is not configured")
	}

	query := `
		SELECT id, title, description, location, salary, company, posted_by, created_at
		FROM jobs
		WHERE id = $1
	`

	var job models.Job
	err := r.DB.QueryRow(query, id).Scan(
		&job.ID,
		&job.Title,
		&job.Description,
		&job.Location,
		&job.Salary,
		&job.Company,
		&job.PostedBy,
		&job.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}

	return &job, nil
}

func (r *JobRepository) GetAllJobs() ([]models.Job, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("database is not configured")
	}

	query := `
		SELECT id, title, description, location, salary, company, posted_by, created_at
		FROM jobs
		ORDER BY created_at DESC
	`

	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]models.Job, 0)
	for rows.Next() {
		var job models.Job
		if err := rows.Scan(
			&job.ID,
			&job.Title,
			&job.Description,
			&job.Location,
			&job.Salary,
			&job.Company,
			&job.PostedBy,
			&job.CreatedAt,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

func (r *JobRepository) DeleteJob(id int) error {
	if r == nil || r.DB == nil {
		return errors.New("database is not configured")
	}

	result, err := r.DB.Exec(`DELETE FROM jobs WHERE id = $1`, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrJobNotFound
	}

	return nil
}

func (r *JobRepository) UpdateJob(id int, input UpdateJobInput) (*models.Job, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("database is not configured")
	}

	query := `
		UPDATE jobs
		SET
			title = COALESCE($2, title),
			description = COALESCE($3, description),
			location = COALESCE($4, location),
			salary = COALESCE($5, salary),
			company = COALESCE($6, company)
		WHERE id = $1
		RETURNING id, title, description, location, salary, company, posted_by, created_at
	`

	var job models.Job
	err := r.DB.QueryRow(
		query,
		id,
		input.Title,
		input.Description,
		input.Location,
		input.Salary,
		input.Company,
	).Scan(
		&job.ID,
		&job.Title,
		&job.Description,
		&job.Location,
		&job.Salary,
		&job.Company,
		&job.PostedBy,
		&job.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}

	return &job, nil
}
