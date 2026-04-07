package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/its-asif/job-portal/internal/models"
	"github.com/its-asif/job-portal/internal/repository"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func TestRegister_ValidRequest_ReturnsCreatedUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := repository.NewUserRepository(db)
	handler := NewUserHandler(repo)

	now := time.Now().UTC()
	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs("Test User", "test@example.com", sqlmock.AnyArg(), "jobseeker").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(1, now))

	payload := map[string]string{
		"name":     "Test User",
		"email":    "test@example.com",
		"password": "secret123",
		"role":     "jobseeker",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	handler.Register(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusCreated, res.Code, res.Body.String())
	}

	var user models.User
	if err := json.Unmarshal(res.Body.Bytes(), &user); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if user.ID != 1 {
		t.Fatalf("expected user ID 1, got %d", user.ID)
	}
	if user.Name != "Test User" {
		t.Fatalf("expected name Test User, got %q", user.Name)
	}
	if user.Email != "test@example.com" {
		t.Fatalf("expected email test@example.com, got %q", user.Email)
	}
	if user.Role != "jobseeker" {
		t.Fatalf("expected role jobseeker, got %q", user.Role)
	}
	if user.Password != "" {
		t.Fatalf("expected password to be omitted in response")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
func TestLogin_ValidRequest_ReturnsToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	t.Setenv("JWT_SECRET", "test-secret")
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	mock.ExpectQuery(`SELECT id, name, email, password, role, created_at\s+FROM users\s+WHERE email = \$1`).
		WithArgs("test@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "password", "role", "created_at"}).
			AddRow(1, "Test User", "test@example.com", string(hashedPassword), "jobseeker", time.Now().UTC()))

	payload := map[string]string{
		"email":    "test@example.com",
		"password": "secret123",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	handler := NewUserHandler(repository.NewUserRepository(db))
	handler.Login(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, res.Code, res.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(res.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	token, ok := resp["token"]
	if !ok || token == "" {
		t.Fatalf("expected token in response")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestRegister_MissingEmail_ReturnsBadRequest(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	handler := NewUserHandler(repository.NewUserRepository(db))
	payload := map[string]string{
		"name":     "No Email",
		"password": "secret123",
		"role":     "jobseeker",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	handler.Register(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusBadRequest, res.Code, res.Body.String())
	}
}

func TestRegister_DuplicateEmail_ReturnsConflict(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	handler := NewUserHandler(repository.NewUserRepository(db))
	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs("Dup User", "dup@example.com", sqlmock.AnyArg(), "jobseeker").
		WillReturnError(&pq.Error{Code: "23505"})

	payload := map[string]string{
		"name":     "Dup User",
		"email":    "dup@example.com",
		"password": "secret123",
		"role":     "jobseeker",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	handler.Register(res, req)

	if res.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusConflict, res.Code, res.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestLogin_WrongPassword_ReturnsUnauthorized(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	t.Setenv("JWT_SECRET", "test-secret")
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	handler := NewUserHandler(repository.NewUserRepository(db))
	mock.ExpectQuery(`SELECT id, name, email, password, role, created_at\s+FROM users\s+WHERE email = \$1`).
		WithArgs("test@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "password", "role", "created_at"}).
			AddRow(1, "Test User", "test@example.com", string(hashedPassword), "jobseeker", time.Now().UTC()))

	payload := map[string]string{
		"email":    "test@example.com",
		"password": "wrong-password",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	handler.Login(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusUnauthorized, res.Code, res.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestLogin_EmailNotFound_ReturnsUnauthorized(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	t.Setenv("JWT_SECRET", "test-secret")
	handler := NewUserHandler(repository.NewUserRepository(db))
	mock.ExpectQuery(`SELECT id, name, email, password, role, created_at\s+FROM users\s+WHERE email = \$1`).
		WithArgs("missing@example.com").
		WillReturnError(sql.ErrNoRows)

	payload := map[string]string{
		"email":    "missing@example.com",
		"password": "secret123",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	handler.Login(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusUnauthorized, res.Code, res.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
