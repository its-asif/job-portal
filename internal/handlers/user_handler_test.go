package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/its-asif/job-portal/internal/models"
	"github.com/its-asif/job-portal/internal/repository"
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
