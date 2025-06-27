package main

import (
	"log"
	"net/http"
	"os"
	"strategic-insight-analyst/handlers"
	"strategic-insight-analyst/utils"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	utils.LoadConfig()

	db, err := utils.InitDB()
	if err != nil {
		log.Fatalf("Failed to connect to Neon: %v", err)
	}
	defer db.Close()

	firebaseApp, err := utils.InitFirebase()
	if err != nil {
		log.Fatalf("Firebase init failed: %v", err)
	}

	documentService := handlers.NewDocumentService(db)
	llmService := handlers.NewLLMService(db)

	r := mux.NewRouter()
	// Register endpoint (no auth)
	r.HandleFunc("/api/register", handlers.RegisterHandler(db)).Methods("POST")
	api := r.PathPrefix("/api").Subrouter()
	api.Use(handlers.AuthMiddleware(firebaseApp, db))

	api.HandleFunc("/documents", documentService.UploadDocument).Methods("POST")
	api.HandleFunc("/documents", documentService.ListDocuments).Methods("GET")
	api.HandleFunc("/documents/{id}", documentService.GetDocument).Methods("GET")
	api.HandleFunc("/documents/{id}", documentService.DeleteDocument).Methods("DELETE")
	api.HandleFunc("/documents/{documentId}/insights", llmService.GenerateInsight).Methods("POST")
	api.HandleFunc("/documents/{documentId}/chat", llmService.ChatWithDocument).Methods("POST")
	api.HandleFunc("/documents/{documentId}/chat/history", llmService.GetChatHistory).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, withCORS(r)))
}
