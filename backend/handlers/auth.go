package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"

	"strategic-insight-analyst/utils" // Replace with your actual path

	firebase "firebase.google.com/go/v4"
)

type contextKey string

const userIDKey contextKey = "userID"

func AuthMiddleware(app *firebase.App, db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/login" || r.URL.Path == "/api/register" {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
				return
			}

			idToken := strings.TrimPrefix(authHeader, "Bearer ")
			idToken = strings.TrimSpace(idToken)
			if idToken == "" {
				http.Error(w, "ID token missing", http.StatusUnauthorized)
				return
			}

			token, err := utils.VerifyIDToken(r.Context(), app, idToken)
			if err != nil {
				http.Error(w, fmt.Sprintf("Invalid token: %v", err), http.StatusUnauthorized)
				return
			}

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

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
