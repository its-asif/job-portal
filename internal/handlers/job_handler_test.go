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
	"github.com/gorilla/mux"
	"github.com/its-asif/job-portal/internal/auth"
	"github.com/its-asif/job-portal/internal/middleware"
	"github.com/its-asif/job-portal/internal/models"
	"github.com/its-asif/job-portal/internal/repository"
)

func TestCreateJob_WithoutAuth_ReturnsUnauthorized(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	handler := NewJobHandler(repository.NewJobRepository(db), repository.NewApplicationRepository(db))

	router := mux.NewRouter()
	router.Handle("/jobs", middleware.AuthMiddleware(middleware.RequireRole("employer")(http.HandlerFunc(handler.CreateJob)))).Methods(http.MethodPost)

	payload := map[string]any{
		"title":       "Backend Engineer",
		"description": "Build APIs",
		"location":    "Dhaka",
		"salary":      100000,
		"company":     "TalentDock",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusUnauthorized, res.Code, res.Body.String())
	}
}

func TestCreateJob_WithAuth_ReturnsCreated(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	t.Setenv("JWT_SECRET", "test-secret")
	token := auth.GenerateToken(7, "employer")
	if token == "" {
		t.Fatalf("expected token to be generated")
	}

	now := time.Now().UTC()
	mock.ExpectQuery(`INSERT INTO jobs`).
		WithArgs("Backend Engineer", "Build APIs", "Dhaka", int64(100000), "TalentDock", 7).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(11, now))

	handler := NewJobHandler(repository.NewJobRepository(db), repository.NewApplicationRepository(db))

	router := mux.NewRouter()
	router.Handle("/jobs", middleware.AuthMiddleware(middleware.RequireRole("employer")(http.HandlerFunc(handler.CreateJob)))).Methods(http.MethodPost)

	payload := map[string]any{
		"title":       "Backend Engineer",
		"description": "Build APIs",
		"location":    "Dhaka",
		"salary":      100000,
		"company":     "TalentDock",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusCreated, res.Code, res.Body.String())
	}

	var job models.Job
	if err := json.Unmarshal(res.Body.Bytes(), &job); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if job.ID != 11 {
		t.Fatalf("expected job ID 11, got %d", job.ID)
	}
	if job.PostedBy != 7 {
		t.Fatalf("expected posted_by 7, got %d", job.PostedBy)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestGetAllJobs_EmptyResult_ReturnsEmptyArray(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, title, description, location, salary, company, posted_by, created_at\s+FROM jobs\s+ORDER BY created_at DESC`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "description", "location", "salary", "company", "posted_by", "created_at"}))

	handler := NewJobHandler(repository.NewJobRepository(db), repository.NewApplicationRepository(db))
	req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	res := httptest.NewRecorder()

	handler.GetAllJobs(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, res.Code, res.Body.String())
	}

	var jobs []models.Job
	if err := json.Unmarshal(res.Body.Bytes(), &jobs); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if jobs == nil {
		t.Fatalf("expected empty array, got null")
	}
	if len(jobs) != 0 {
		t.Fatalf("expected empty array, got %d jobs", len(jobs))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestGetJobByID_InvalidID_ReturnsBadRequest(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	handler := NewJobHandler(repository.NewJobRepository(db), repository.NewApplicationRepository(db))
	router := mux.NewRouter()
	router.HandleFunc("/jobs/{id}", handler.GetJobByID).Methods(http.MethodGet)

	req := httptest.NewRequest(http.MethodGet, "/jobs/abc", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusBadRequest, res.Code, res.Body.String())
	}
}

func TestGetJobByID_NotFound_ReturnsNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, title, description, location, salary, company, posted_by, created_at\s+FROM jobs\s+WHERE id = \$1`).
		WithArgs(999).
		WillReturnError(sql.ErrNoRows)

	handler := NewJobHandler(repository.NewJobRepository(db), repository.NewApplicationRepository(db))
	router := mux.NewRouter()
	router.HandleFunc("/jobs/{id}", handler.GetJobByID).Methods(http.MethodGet)

	req := httptest.NewRequest(http.MethodGet, "/jobs/999", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusNotFound, res.Code, res.Body.String())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
