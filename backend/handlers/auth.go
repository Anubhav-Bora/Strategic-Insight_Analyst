package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"

	"strategic-insight-analyst/utils" // ✅ Replace this with your actual module path

	firebase "firebase.google.com/go/v4"
)

// Define a custom type for context key to avoid collisions
type contextKey string

const userIDKey contextKey = "userID"

// AuthMiddleware verifies Firebase ID token and checks if user exists in DB
func AuthMiddleware(app *firebase.App, db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow public routes without authentication
			if r.URL.Path == "/api/login" || r.URL.Path == "/api/register" {
				next.ServeHTTP(w, r)
				return
			}

			// Get Authorization header
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			// Extract the token
			idToken := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			if idToken == "" {
				http.Error(w, "ID token missing", http.StatusUnauthorized)
				return
			}

			// Verify the token using Firebase
			token, err := utils.VerifyIDToken(r.Context(), app, idToken)
			if err != nil {
				http.Error(w, fmt.Sprintf("Invalid ID token: %v", err), http.StatusUnauthorized)
				return
			}

			// Check if user exists in the database
			var userID string
			err = db.QueryRowContext(r.Context(), "SELECT id FROM users WHERE id = $1", token.UID).Scan(&userID)
			if err != nil {
				if err == sql.ErrNoRows {
					http.Error(w, "User not registered", http.StatusUnauthorized)
					return
				}
				log.Printf("Database error: %v", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}

			// Add user ID to the request context
			ctx := context.WithValue(r.Context(), userIDKey, token.UID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
