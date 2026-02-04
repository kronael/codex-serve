package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateSecret generates a cryptographically secure random secret
func GenerateSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secret: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GenerateToken creates a JWT token with the given subject and expiration
func GenerateToken(secret, subject string, expiration time.Duration) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("jwt secret is empty")
	}

	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signed, nil
}

// JWTMiddleware validates JWT tokens from Authorization header
func JWTMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if secret == "" {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				WriteError(w, &APIError{
					Code:    ErrUnauthorized,
					Message: "missing authorization header",
					Status:  http.StatusUnauthorized,
				})
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				WriteError(w, &APIError{
					Code:    ErrUnauthorized,
					Message: "invalid authorization header format",
					Status:  http.StatusUnauthorized,
				})
				return
			}

			tokenString := parts[1]
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(secret), nil
			})

			if err != nil {
				WriteError(w, &APIError{
					Code:    ErrUnauthorized,
					Message: fmt.Sprintf("invalid token: %v", err),
					Status:  http.StatusUnauthorized,
				})
				return
			}

			if !token.Valid {
				WriteError(w, &APIError{
					Code:    ErrUnauthorized,
					Message: "token is not valid",
					Status:  http.StatusUnauthorized,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
