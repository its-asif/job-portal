package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/its-asif/job-portal/internal/middleware"
	"github.com/its-asif/job-portal/internal/models"
	"github.com/its-asif/job-portal/internal/repository"
	"github.com/redis/go-redis/v9"
)

type JobHandler struct {
	Repo            *repository.JobRepository
	ApplicationRepo *repository.ApplicationRepository
	UserRepo        *repository.UserRepository
	OutboxRepo      *repository.OutboxRepository
	QueuePublisher  interface {
		Publish(queueName, message string) error
	}
	RedisClient *redis.Client
}

type createJobRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Salary      int64  `json:"salary"`
	Company     string `json:"company"`
}

type updateJobRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Location    *string `json:"location"`
	Salary      *int64  `json:"salary"`
	Company     *string `json:"company"`
}

type updateApplicationStatusRequest struct {
	Status string `json:"status"`
}

func NewJobHandler(repo *repository.JobRepository, applicationRepo *repository.ApplicationRepository) *JobHandler {
	return &JobHandler{
		Repo:            repo,
		ApplicationRepo: applicationRepo,
	}
}

func (h *JobHandler) SetRedisClient(client *redis.Client) {
	h.RedisClient = client
}

func (h *JobHandler) SetUserRepo(repo *repository.UserRepository) {
	h.UserRepo = repo
}

func (h *JobHandler) SetOutboxRepo(repo *repository.OutboxRepository) {
	h.OutboxRepo = repo
}

func (h *JobHandler) SetQueuePublisher(publisher interface {
	Publish(queueName, message string) error
}) {
	h.QueuePublisher = publisher
}

// CreateJob godoc
// @Summary Create job
// @Description Create a job post. Employer role required.
// @Tags jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body models.CreateJobRequest true "Create job payload"
// @Success 201 {object} models.Job
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /jobs [post]
func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	if h.Repo == nil {
		respondWithError(w, http.StatusInternalServerError, "database is not configured")
		return
	}

	var req createJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	req.Location = strings.TrimSpace(req.Location)
	req.Company = strings.TrimSpace(req.Company)

	if req.Title == "" || req.Description == "" || req.Location == "" || req.Company == "" {
		respondWithError(w, http.StatusBadRequest, "title, description, location, and company are required")
		return
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	job := &models.Job{
		Title:       req.Title,
		Description: req.Description,
		Location:    req.Location,
		Salary:      req.Salary,
		Company:     req.Company,
		PostedBy:    claims.UserID,
	}

	if err := h.Repo.CreateJob(job); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to create job")
		return
	}

	if h.RedisClient != nil {
		_ = h.RedisClient.Del(r.Context(), "jobs:all").Err()
	}

	respondWithJSON(w, http.StatusCreated, job)
}

// GetAllJobs godoc
// @Summary List jobs
// @Description Get all jobs.
// @Tags jobs
// @Produce json
// @Success 200 {array} models.Job
// @Failure 500 {object} models.ErrorResponse
// @Router /jobs [get]
func (h *JobHandler) GetAllJobs(w http.ResponseWriter, r *http.Request) {
	if h.Repo == nil {
		respondWithError(w, http.StatusInternalServerError, "database is not configured")
		return
	}

	if h.RedisClient != nil {
		if cached, err := h.RedisClient.Get(r.Context(), "jobs:all").Result(); err == nil {
			var jobs []models.Job
			if unmarshalErr := json.Unmarshal([]byte(cached), &jobs); unmarshalErr == nil {
				respondWithJSON(w, http.StatusOK, jobs)
				return
			}
		}
	}

	jobs, err := h.Repo.GetAllJobs()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to fetch jobs")
		return
	}

	if jobs == nil {
		jobs = make([]models.Job, 0)
	}

	if h.RedisClient != nil {
		if payload, marshalErr := json.Marshal(jobs); marshalErr == nil {
			_ = h.RedisClient.Set(r.Context(), "jobs:all", payload, 2*time.Minute).Err()
		}
	}

	respondWithJSON(w, http.StatusOK, jobs)
}

// ApplyToJob godoc
// @Summary Apply to a job
// @Description Apply to a job by ID. Jobseeker role required.
// @Tags applications
// @Produce json
// @Security BearerAuth
// @Param id path int true "Job ID"
// @Success 201 {object} models.Application
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /jobs/{id}/apply [post]
func (h *JobHandler) ApplyToJob(w http.ResponseWriter, r *http.Request) {
	if h.ApplicationRepo == nil {
		respondWithError(w, http.StatusInternalServerError, "database is not configured")
		return
	}

	jobID, err := parseJobID(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	if h.Repo == nil {
		respondWithError(w, http.StatusInternalServerError, "database is not configured")
		return
	}

	if _, err := h.Repo.GetJobByID(jobID); err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			respondWithError(w, http.StatusNotFound, "job not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to apply to job")
		return
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	application, err := h.ApplicationRepo.CreateApplication(jobID, claims.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyApplied) {
			respondWithError(w, http.StatusConflict, "already applied to this job")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to apply to job")
		return
	}

	h.publishNewApplicationEvent(r.Context(), application)

	respondWithJSON(w, http.StatusCreated, application)
}

func (h *JobHandler) publishNewApplicationEvent(ctx context.Context, application *models.Application) {
	if application == nil {
		return
	}

	applicantEmail := ""
	if h.UserRepo != nil {
		if user, err := h.UserRepo.GetUserByID(application.UserID); err == nil {
			applicantEmail = user.Email
		}
	}

	payload := map[string]interface{}{
		"type":            "new_application",
		"job_id":          application.JobID,
		"applicant_id":    application.UserID,
		"applicant_email": applicantEmail,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return
	}

	if h.QueuePublisher != nil {
		if err := h.QueuePublisher.Publish("notifications", string(payloadBytes)); err == nil {
			return
		}
	}

	if h.OutboxRepo != nil {
		_ = h.OutboxRepo.CreateEvent("new_application", "notifications", payloadBytes)
	}
}

// GetApplicationsByJobID godoc
// @Summary List applications for a job
// @Description Get all applications submitted for a specific job.
// @Tags applications
// @Produce json
// @Param id path int true "Job ID"
// @Success 200 {array} models.Application
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /jobs/{id}/applications [get]
func (h *JobHandler) GetApplicationsByJobID(w http.ResponseWriter, r *http.Request) {
	if h.ApplicationRepo == nil {
		respondWithError(w, http.StatusInternalServerError, "database is not configured")
		return
	}

	jobID, err := parseJobID(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	applications, err := h.ApplicationRepo.GetApplicationsByJobID(jobID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to fetch applications")
		return
	}

	if applications == nil {
		applications = make([]models.Application, 0)
	}

	respondWithJSON(w, http.StatusOK, applications)
}

// GetJobByID godoc
// @Summary Get job by ID
// @Description Get one job by ID.
// @Tags jobs
// @Produce json
// @Param id path int true "Job ID"
// @Success 200 {object} models.Job
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /jobs/{id} [get]
func (h *JobHandler) GetJobByID(w http.ResponseWriter, r *http.Request) {
	if h.Repo == nil {
		respondWithError(w, http.StatusInternalServerError, "database is not configured")
		return
	}

	jobID, err := parseJobID(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	cacheKey := fmt.Sprintf("jobs:%d", jobID)
	if h.RedisClient != nil {
		if cached, cacheErr := h.RedisClient.Get(r.Context(), cacheKey).Result(); cacheErr == nil {
			var cachedJob models.Job
			if unmarshalErr := json.Unmarshal([]byte(cached), &cachedJob); unmarshalErr == nil {
				respondWithJSON(w, http.StatusOK, &cachedJob)
				return
			}
		}
	}

	job, err := h.Repo.GetJobByID(jobID)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			respondWithError(w, http.StatusNotFound, "job not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to fetch job")
		return
	}

	if h.RedisClient != nil {
		if payload, marshalErr := json.Marshal(job); marshalErr == nil {
			_ = h.RedisClient.Set(r.Context(), cacheKey, payload, 2*time.Minute).Err()
		}
	}

	respondWithJSON(w, http.StatusOK, job)
}

// DeleteJob godoc
// @Summary Delete job
// @Description Delete a job by ID. Employer role required.
// @Tags jobs
// @Produce json
// @Security BearerAuth
// @Param id path int true "Job ID"
// @Success 204
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /jobs/{id} [delete]
func (h *JobHandler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	if h.Repo == nil {
		respondWithError(w, http.StatusInternalServerError, "database is not configured")
		return
	}

	jobID, err := parseJobID(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	if err := h.Repo.DeleteJob(jobID); err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			respondWithError(w, http.StatusNotFound, "job not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to delete job")
		return
	}

	if h.RedisClient != nil {
		_ = h.RedisClient.Del(r.Context(), "jobs:all", fmt.Sprintf("jobs:%d", jobID)).Err()
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateJob godoc
// @Summary Update job
// @Description Update one or more job fields by ID.
// @Tags jobs
// @Accept json
// @Produce json
// @Param id path int true "Job ID"
// @Param payload body models.UpdateJobRequest true "Update job payload"
// @Success 200 {object} models.Job
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /jobs/{id} [put]
func (h *JobHandler) UpdateJob(w http.ResponseWriter, r *http.Request) {
	if h.Repo == nil {
		respondWithError(w, http.StatusInternalServerError, "database is not configured")
		return
	}

	jobID, err := parseJobID(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid job id")
		return
	}

	var req updateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	trimPtr(req.Title)
	trimPtr(req.Description)
	trimPtr(req.Location)
	trimPtr(req.Company)

	if req.Title == nil && req.Description == nil && req.Location == nil && req.Salary == nil && req.Company == nil {
		respondWithError(w, http.StatusBadRequest, "at least one field is required")
		return
	}

	updatedJob, err := h.Repo.UpdateJob(jobID, repository.UpdateJobInput{
		Title:       req.Title,
		Description: req.Description,
		Location:    req.Location,
		Salary:      req.Salary,
		Company:     req.Company,
	})
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			respondWithError(w, http.StatusNotFound, "job not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to update job")
		return
	}

	if h.RedisClient != nil {
		_ = h.RedisClient.Del(r.Context(), "jobs:all", fmt.Sprintf("jobs:%d", jobID)).Err()
	}

	respondWithJSON(w, http.StatusOK, updatedJob)
}

// UpdateApplicationStatus godoc
// @Summary Update application status
// @Description Update application status to reviewed, accepted, or rejected. Employer role required.
// @Tags applications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Application ID"
// @Param payload body models.UpdateApplicationStatusRequest true "Update application status payload"
// @Success 200 {object} models.Application
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /applications/{id}/status [put]
func (h *JobHandler) UpdateApplicationStatus(w http.ResponseWriter, r *http.Request) {
	if h.ApplicationRepo == nil {
		respondWithError(w, http.StatusInternalServerError, "database is not configured")
		return
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	applicationID, err := parseApplicationID(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid application id")
		return
	}

	var req updateApplicationStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Status = strings.TrimSpace(strings.ToLower(req.Status))
	if req.Status != "reviewed" && req.Status != "accepted" && req.Status != "rejected" {
		respondWithError(w, http.StatusBadRequest, "status must be reviewed, accepted, or rejected")
		return
	}

	application, err := h.ApplicationRepo.UpdateApplicationStatus(applicationID, claims.UserID, req.Status)
	if err != nil {
		if errors.Is(err, repository.ErrApplicationNotFound) {
			respondWithError(w, http.StatusNotFound, "application not found")
			return
		}
		if errors.Is(err, repository.ErrEmployerNotAllowed) {
			respondWithError(w, http.StatusForbidden, "forbidden")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to update application status")
		return
	}

	respondWithJSON(w, http.StatusOK, application)
}

func parseJobID(r *http.Request) (int, error) {
	jobIDParam := mux.Vars(r)["id"]
	jobID, err := strconv.Atoi(jobIDParam)
	if err != nil {
		return 0, err
	}

	return jobID, nil
}

func parseApplicationID(r *http.Request) (int, error) {
	applicationIDParam := mux.Vars(r)["id"]
	applicationID, err := strconv.Atoi(applicationIDParam)
	if err != nil {
		return 0, err
	}

	return applicationID, nil
}

func trimPtr(value *string) {
	if value == nil {
		return
	}
	trimmed := strings.TrimSpace(*value)
	*value = trimmed
}
