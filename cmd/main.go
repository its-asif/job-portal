package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/its-asif/job-portal/db"
	"github.com/its-asif/job-portal/internal/handlers"
	"github.com/its-asif/job-portal/internal/repository"
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
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello world"))
	}).Methods(http.MethodGet)

	userRepo := repository.NewUserRepository(dbConn)
	userHandler := handlers.NewUserHandler(userRepo)
	r.HandleFunc("/register", userHandler.Register).Methods(http.MethodPost)
	r.HandleFunc("/login", userHandler.Login).Methods(http.MethodPost)

	jobRepo := repository.NewJobRepository(dbConn)
	jobHandler := handlers.NewJobHandler(jobRepo)
	r.HandleFunc("/jobs", jobHandler.CreateJob).Methods(http.MethodPost)
	r.HandleFunc("/jobs", jobHandler.GetAllJobs).Methods(http.MethodGet)
	r.HandleFunc("/jobs/{id}", jobHandler.GetJobByID).Methods(http.MethodGet)
	r.HandleFunc("/jobs/{id}", jobHandler.DeleteJob).Methods(http.MethodDelete)
	r.HandleFunc("/jobs/{id}", jobHandler.UpdateJob).Methods(http.MethodPut)

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
