package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/its-asif/job-portal/internal/models"
	"github.com/its-asif/job-portal/internal/repository"
)

type JobHandler struct {
	Repo *repository.JobRepository
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

func NewJobHandler(repo *repository.JobRepository) *JobHandler {
	return &JobHandler{Repo: repo}
}

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

	job := &models.Job{
		Title:       req.Title,
		Description: req.Description,
		Location:    req.Location,
		Salary:      req.Salary,
		Company:     req.Company,
		PostedBy:    1,
	}

	if err := h.Repo.CreateJob(job); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to create job")
		return
	}

	respondWithJSON(w, http.StatusCreated, job)
}

func (h *JobHandler) GetAllJobs(w http.ResponseWriter, r *http.Request) {
	if h.Repo == nil {
		respondWithError(w, http.StatusInternalServerError, "database is not configured")
		return
	}

	jobs, err := h.Repo.GetAllJobs()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to fetch jobs")
		return
	}

	if jobs == nil {
		jobs = make([]models.Job, 0)
	}

	respondWithJSON(w, http.StatusOK, jobs)
}

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

	job, err := h.Repo.GetJobByID(jobID)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			respondWithError(w, http.StatusNotFound, "job not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to fetch job")
		return
	}

	respondWithJSON(w, http.StatusOK, job)
}

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

	w.WriteHeader(http.StatusNoContent)
}

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

	respondWithJSON(w, http.StatusOK, updatedJob)
}

func parseJobID(r *http.Request) (int, error) {
	jobIDParam := mux.Vars(r)["id"]
	jobID, err := strconv.Atoi(jobIDParam)
	if err != nil {
		return 0, err
	}

	return jobID, nil
}

func trimPtr(value *string) {
	if value == nil {
		return
	}
	trimmed := strings.TrimSpace(*value)
	*value = trimmed
}
