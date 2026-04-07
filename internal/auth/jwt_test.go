package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAndValidateToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	token := GenerateToken(42, "employer")
	if token == "" {
		t.Fatalf("expected token to be generated")
	}

	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("expected token to validate, got error: %v", err)
	}

	if claims.UserID != 42 {
		t.Fatalf("expected user_id 42, got %d", claims.UserID)
	}
	if claims.Role != "employer" {
		t.Fatalf("expected role employer, got %s", claims.Role)
	}
}

func TestValidateToken_ExpiredTokenRejected(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	expiredClaims := &Claims{
		UserID: 99,
		Role:   "jobseeker",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	tokenString, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("failed to sign expired token: %v", err)
	}

	if _, err := ValidateToken(tokenString); err == nil {
		t.Fatalf("expected expired token validation to fail")
	}
}
