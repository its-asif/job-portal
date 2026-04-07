package models

// TokenResponse contains JWT after successful login.
type TokenResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// ErrorResponse is a standard API error payload.
type ErrorResponse struct {
	Error string `json:"error" example:"invalid request body"`
}
