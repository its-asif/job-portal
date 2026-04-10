package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/its-asif/job-portal/internal/auth"
	"github.com/its-asif/job-portal/internal/middleware"
	"github.com/its-asif/job-portal/internal/models"
	"github.com/its-asif/job-portal/internal/repository"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	Repo        *repository.UserRepository
	RedisClient *redis.Client
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

func (h *UserHandler) SetRedisClient(client *redis.Client) {
	h.RedisClient = client
}

// Register godoc
// @Summary Register a new user
// @Description Register a new employer or jobseeker account.
// @Tags auth
// @Accept json
// @Produce json
// @Param payload body models.RegisterRequest true "Register payload"
// @Success 201 {object} models.User
// @Failure 400 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /register [post]
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

	if req.Role != "employer" && req.Role != "jobseeker" {
		respondWithError(w, http.StatusBadRequest, "role must be employer or jobseeker")
		return
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

// Login godoc
// @Summary Login user
// @Description Validate credentials and return JWT token.
// @Tags auth
// @Accept json
// @Produce json
// @Param payload body models.LoginRequest true "Login payload"
// @Success 200 {object} models.TokenResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /login [post]
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

	token := auth.GenerateToken(user.ID, user.Role)
	if token == "" {
		respondWithError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"token": token})
}

// GetAllUsers godoc
// @Summary List users
// @Description Get all registered users.
// @Tags users
// @Produce json
// @Success 200 {array} models.User
// @Failure 500 {object} models.ErrorResponse
// @Router /users [get]
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

// GetUserByID godoc
// @Summary Get user by ID
// @Description Get a single user by ID.
// @Tags users
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} models.User
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /users/{id} [get]
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

// GetMe godoc
// @Summary Get current user profile
// @Description Return profile of currently authenticated user.
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.User
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /me [get]
func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	if h.Repo == nil {
		respondWithError(w, http.StatusInternalServerError, "database is not configured")
		return
	}

	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.Repo.GetUserByID(claims.UserID)
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

// Logout godoc
// @Summary Logout user
// @Description Revoke current JWT by adding it to Redis blacklist.
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string
// @Failure 401 {object} models.ErrorResponse
// @Failure 503 {object} models.ErrorResponse
// @Router /logout [post]
func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if h.RedisClient == nil {
		respondWithError(w, http.StatusServiceUnavailable, "redis is not configured")
		return
	}

	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		respondWithError(w, http.StatusUnauthorized, "invalid authorization header")
		return
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		respondWithError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}

	claims, err := auth.ValidateToken(token)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl <= 0 {
		ttl = time.Second
	}

	if err := h.RedisClient.Set(r.Context(), "blacklist:"+token, "1", ttl).Err(); err != nil {
		respondWithError(w, http.StatusServiceUnavailable, "failed to revoke token")
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "logout successful"})
}

func parseUserID(r *http.Request) (int, error) {
	userIDParam := mux.Vars(r)["id"]
	return strconv.Atoi(userIDParam)
}
