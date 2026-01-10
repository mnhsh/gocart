package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

type contextKey string

const (
	UserIDContextKey   contextKey = "userID"
	UserRoleContextKey contextKey = "userRole"
)

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, string, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.Nil, "", err
	}

	if !token.Valid {
		return uuid.Nil, "", fmt.Errorf("invalid token")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, "", err
	}

	if claims.Role == "" {
		return uuid.Nil, "", fmt.Errorf("missing role in token")
	}

	return userID, claims.Role, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("auth header missing")
	}
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("invalid authorization header format")
	}
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if token == "" {
		return "", fmt.Errorf("invalid authorization header format")
	}
	return token, nil
}

func GetUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(UserIDContextKey).(uuid.UUID)
	return userID, ok
}

func GetUserRoleFromContext(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(UserRoleContextKey).(string)
	return role, ok
}
