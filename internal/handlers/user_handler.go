package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/its-asif/job-portal/internal/models"
	"github.com/its-asif/job-portal/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	Repo *repository.UserRepository
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func NewUserHandler(repo *repository.UserRepository) *UserHandler {
	return &UserHandler{Repo: repo}
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	if h.Repo == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database is not configured"})
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)
	req.Name = strings.TrimSpace(req.Name)
	req.Role = strings.TrimSpace(req.Role)

	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email and password are required"})
		return
	}

	if req.Role == "" {
		req.Role = "jobseeker"
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
		return
	}

	user := &models.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hashedPassword),
		Role:     req.Role,
	}

	if err := h.Repo.CreateUser(user); err != nil {
		if errors.Is(err, repository.ErrDuplicateEmail) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "email already exists"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
		return
	}

	user.Password = ""
	writeJSON(w, http.StatusCreated, user)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	if h.Repo == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database is not configured"})
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)
	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email and password are required"})
		return
	}

	user, err := h.Repo.GetUserByEmail(req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to login"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid email or password"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "login successful"})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
