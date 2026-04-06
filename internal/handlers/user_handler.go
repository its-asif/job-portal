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
		respondWithError(w, http.StatusInternalServerError, "database is not configured")
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)
	req.Name = strings.TrimSpace(req.Name)
	req.Role = strings.TrimSpace(req.Role)

	if req.Email == "" || req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	if req.Role == "" {
		req.Role = "jobseeker"
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to hash password")
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
			respondWithError(w, http.StatusConflict, "email already exists")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	user.Password = ""
	respondWithJSON(w, http.StatusCreated, user)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	if h.Repo == nil {
		respondWithError(w, http.StatusInternalServerError, "database is not configured")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)
	if req.Email == "" || req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := h.Repo.GetUserByEmail(req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			respondWithError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to login")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "login successful"})
}

func (h *UserHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	if h.Repo == nil {
		respondWithError(w, http.StatusInternalServerError, "database is not configured")
		return
	}

	users, err := h.Repo.GetAllUsers()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to fetch users")
		return
	}

	if users == nil {
		users = make([]models.User, 0)
	}

	for i := range users {
		users[i].Password = ""
	}

	respondWithJSON(w, http.StatusOK, users)
}

func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	if h.Repo == nil {
		respondWithError(w, http.StatusInternalServerError, "database is not configured")
		return
	}

	userID, err := parseUserID(r)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	user, err := h.Repo.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			respondWithError(w, http.StatusNotFound, "user not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to fetch user")
		return
	}

	user.Password = ""
	respondWithJSON(w, http.StatusOK, user)
}

func parseUserID(r *http.Request) (int, error) {
	userIDParam := mux.Vars(r)["id"]
	return strconv.Atoi(userIDParam)
}
