package main

import (
	"log"
	"net/http"
	"os"
	"strategic-insight-analyst/handlers"
	"strategic-insight-analyst/utils"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	utils.LoadConfig()

	// Initialize Neon DB
	db, err := utils.InitDB()
	if err != nil {
		log.Fatalf("❌ Failed to connect to Neon: %v", err)
	}
	defer db.Close()

	// Initialize Firebase
	firebaseApp, err := utils.InitFirebase()
	if err != nil {
		log.Fatalf("❌ Firebase init failed: %v", err)
	}

	// Initialize services
	documentService := handlers.NewDocumentService(db)
	llmService := handlers.NewLLMService(db)

	// Router setup
	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()
	api.Use(handlers.AuthMiddleware(firebaseApp, db)) // Pass db to middleware

	// Document routes
	api.HandleFunc("/documents", documentService.UploadDocument).Methods("POST")
	api.HandleFunc("/documents", documentService.ListDocuments).Methods("GET")
	api.HandleFunc("/documents/{id}", documentService.GetDocument).Methods("GET")
	api.HandleFunc("/documents/{id}", documentService.DeleteDocument).Methods("DELETE")

	// LLM routes
	api.HandleFunc("/documents/{documentId}/insights", llmService.GenerateInsight).Methods("POST")
	api.HandleFunc("/documents/{documentId}/chat", llmService.ChatWithDocument).Methods("POST")
	api.HandleFunc("/documents/{documentId}/chat", llmService.GetChatHistory).Methods("GET")

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("🚀 Server running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, cors.Default().Handler(r)))
}
