// @title Job Portal API
// @version 1.0
// @description Job portal backend API built with Go, Gorilla Mux, and PostgreSQL.
// @host localhost:8080
// @BasePath /
// @schemes http
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/its-asif/job-portal/docs"
	"github.com/gorilla/mux"
	"github.com/its-asif/job-portal/db"
	"github.com/its-asif/job-portal/internal/handlers"
	"github.com/its-asif/job-portal/internal/middleware"
	"github.com/its-asif/job-portal/internal/repository"
	httpSwagger "github.com/swaggo/http-swagger"
)

func main() {
	loadEnvFile(".env")

	dbConn, err := db.Connect()
	if err != nil {
		log.Printf("database ping failed: %v", err)
	} else {
		defer func() {
			if closeErr := dbConn.Close(); closeErr != nil {
				log.Printf("failed to close db connection: %v", closeErr)
			}
		}()
		log.Println("database connected successfully")
	}

	r := mux.NewRouter()
	r.PathPrefix("/docs/").Handler(httpSwagger.WrapHandler)
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello world"))
	}).Methods(http.MethodGet)

	userRepo := repository.NewUserRepository(dbConn)
	userHandler := handlers.NewUserHandler(userRepo)
	r.HandleFunc("/register", userHandler.Register).Methods(http.MethodPost)
	r.HandleFunc("/login", userHandler.Login).Methods(http.MethodPost)
	r.HandleFunc("/users", userHandler.GetAllUsers).Methods(http.MethodGet)
	r.HandleFunc("/users/{id}", userHandler.GetUserByID).Methods(http.MethodGet)

	jobRepo := repository.NewJobRepository(dbConn)
	applicationRepo := repository.NewApplicationRepository(dbConn)
	jobHandler := handlers.NewJobHandler(jobRepo, applicationRepo)
	r.HandleFunc("/jobs", jobHandler.GetAllJobs).Methods(http.MethodGet)
	r.HandleFunc("/jobs/{id}", jobHandler.GetJobByID).Methods(http.MethodGet)
	r.HandleFunc("/jobs/{id}", jobHandler.UpdateJob).Methods(http.MethodPut)
	r.HandleFunc("/jobs/{id}/applications", jobHandler.GetApplicationsByJobID).Methods(http.MethodGet)

	protected := r.PathPrefix("/").Subrouter()
	protected.Use(middleware.AuthMiddleware)
	protected.HandleFunc("/me", userHandler.GetMe).Methods(http.MethodGet)

	employerOnly := protected.PathPrefix("/").Subrouter()
	employerOnly.Use(middleware.RequireRole("employer"))
	employerOnly.HandleFunc("/jobs", jobHandler.CreateJob).Methods(http.MethodPost)
	employerOnly.HandleFunc("/jobs/{id}", jobHandler.DeleteJob).Methods(http.MethodDelete)
	employerOnly.HandleFunc("/applications/{id}/status", jobHandler.UpdateApplicationStatus).Methods(http.MethodPut)

	jobseekerOnly := protected.PathPrefix("/").Subrouter()
	jobseekerOnly.Use(middleware.RequireRole("jobseeker"))
	jobseekerOnly.HandleFunc("/jobs/{id}/apply", jobHandler.ApplyToJob).Methods(http.MethodPost)

	log.Println("server started on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func loadEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		if key == "" {
			continue
		}

		if setErr := os.Setenv(key, value); setErr != nil {
			log.Printf("failed to set env var %s: %v", key, setErr)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("failed to read env file %s: %v", path, err)
	}

	if os.Getenv("DB_URL") == "" {
		log.Printf("DB_URL is empty, check %s", path)
	}
}
