package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"strategic-insight-analyst/utils"
	"firebase.google.com/go/v4"
)

// Middleware that verifies token and checks DB user existence
func AuthMiddleware(app *firebase.App) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow login and register without token
			if r.URL.Path == "/api/login" || r.URL.Path == "/api/register" {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header missing", http.StatusUnauthorized)
				return
			}

			idToken := strings.TrimPrefix(authHeader, "Bearer ")
			if idToken == "" {
				http.Error(w, "ID token missing", http.StatusUnauthorized)
				return
			}

			token, err := utils.VerifyIDToken(r.Context(), app, idToken)
			if err != nil {
				http.Error(w, fmt.Sprintf("Invalid ID token: %v", err), http.StatusUnauthorized)
				return
			}

			// Check if user exists in DB
			db, err := utils.GetDBConnection()
			if err != nil {
				http.Error(w, "Database connection error", http.StatusInternalServerError)
				return
			}
			defer db.Close()

			var userID string
			err = db.QueryRowContext(r.Context(), "SELECT id FROM users WHERE id = $1", token.UID).Scan(&userID)
			if err != nil {
				if err == sql.ErrNoRows {
					http.Error(w, "User not registered", http.StatusUnauthorized)
					return
				}
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}

			// Add userID to request context
			ctx := context.WithValue(r.Context(), "userID", token.UID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Login handles user login with Firebase
func Login(w http.ResponseWriter, r *http.Request) {
	var request struct {
		IDToken string `json:"idToken"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Initialize Firebase app
	app, err := utils.InitFirebase()
	if err != nil {
		http.Error(w, "Firebase initialization error", http.StatusInternalServerError)
		return
	}

	// Verify the ID token
	token, err := utils.VerifyIDToken(r.Context(), app, request.IDToken)
	if err != nil {
		http.Error(w, "Invalid ID token", http.StatusUnauthorized)
		return
	}

	// Check if user exists in database
	db, err := utils.GetDBConnection()
	if err != nil {
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	var userID string
	err = db.QueryRowContext(r.Context(), "SELECT id FROM users WHERE id = $1", token.UID).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not registered", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Login successful",
		"userId":  token.UID,
	})
}
