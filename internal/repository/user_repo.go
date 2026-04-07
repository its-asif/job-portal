package repository

import (
	"database/sql"
	"errors"

	"github.com/its-asif/job-portal/internal/models"
	"github.com/lib/pq"
)

var (
	ErrDuplicateEmail = errors.New("email already exists")
	ErrUserNotFound   = errors.New("user not found")
)

type UserRepository struct {
	DB *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) CreateUser(user *models.User) error {
	if r == nil || r.DB == nil {
		return errors.New("database is not configured")
	}

	query := `
		INSERT INTO users (name, email, password, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

	err := r.DB.QueryRow(query, user.Name, user.Email, user.Password, user.Role).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateEmail
		}
		return err
	}

	return nil
}

func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("database is not configured")
	}

	query := `
		SELECT id, name, email, password, role, created_at
		FROM users
		WHERE email = $1
	`

	var user models.User
	err := r.DB.QueryRow(query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}



func (r *UserRepository) GetUserByID(id int) (*models.User, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("database is not configured")
	}

	query := `
		SELECT id, name, email, password, role, created_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := r.DB.QueryRow(query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetAllUsers() ([]models.User, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("database is not configured")
	}

	query := `
		SELECT id, name, email, password, role, created_at
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]models.User, 0)
	for rows.Next() {
		var user models.User
		if err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.Password,
			&user.Role,
			&user.CreatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
