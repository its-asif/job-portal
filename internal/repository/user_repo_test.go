package repository

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/its-asif/job-portal/internal/models"
	_ "github.com/lib/pq"
)

func TestUserRepository_CreateAndGetByEmail(t *testing.T) {
	testDBURL := os.Getenv("TEST_DB_URL")
	if testDBURL == "" {
		t.Skip("needs db: set TEST_DB_URL to run integration tests")
	}

	db, err := sql.Open("postgres", testDBURL)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping test db: %v", err)
	}

	repo := NewUserRepository(db)
	email := "itest_" + time.Now().UTC().Format("20060102150405.000000") + "@example.com"
	user := &models.User{
		Name:     "Integration User",
		Email:    email,
		Password: "hashed-password",
		Role:     "jobseeker",
	}

	if err := repo.CreateUser(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM users WHERE email = $1", email)
	})

	if user.ID == 0 {
		t.Fatalf("expected created user ID to be set")
	}

	fetched, err := repo.GetUserByEmail(email)
	if err != nil {
		t.Fatalf("failed to fetch user by email: %v", err)
	}
	if fetched.Email != email {
		t.Fatalf("expected email %s, got %s", email, fetched.Email)
	}
	if fetched.Role != "jobseeker" {
		t.Fatalf("expected role jobseeker, got %s", fetched.Role)
	}
}
