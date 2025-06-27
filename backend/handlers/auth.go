package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"strategic-insight-analyst/utils" 

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


func RegisterHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		type reqBody struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		}
		var body reqBody
		err := json.NewDecoder(r.Body).Decode(&body)
		log.Printf("[REGISTER] Received: id=%s, email=%s", body.ID, body.Email)
		if err != nil || body.ID == "" || body.Email == "" {
			log.Printf("[REGISTER] Invalid request body: err=%v, id=%s, email=%s", err, body.ID, body.Email)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		_, err = db.ExecContext(r.Context(), "INSERT INTO users (id, email) VALUES ($1, $2) ON CONFLICT (id) DO NOTHING", body.ID, body.Email)
		if err != nil {
			log.Printf("[REGISTER] Failed to register user: %v", err)
			http.Error(w, "Failed to register user", http.StatusInternalServerError)
			return
		}

		log.Printf("[REGISTER] User registered successfully: id=%s, email=%s", body.ID, body.Email)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("User registered successfully"))
	}
}
