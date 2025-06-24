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
	// Initialize configuration
	utils.LoadConfig()

	// Initialize database connection
	db, err := utils.InitDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Firebase app
	firebaseApp, err := utils.InitFirebase()
	if err != nil {
		log.Fatalf("Failed to initialize Firebase: %v", err)
	}

	// Initialize Google Cloud Storage client
	storageClient, err := utils.InitStorage()
	if err != nil {
		log.Fatalf("Failed to initialize Google Cloud Storage: %v", err)
	}

	// Initialize services
	documentService := handlers.NewDocumentService(db, storageClient)
	llmService := handlers.NewLLMService(db)

	// Set up router
	r := mux.NewRouter()

	// API routes
	api := r.PathPrefix("/api").Subrouter()
	api.Use(handlers.AuthMiddleware(firebaseApp))

	// Auth routes
	r.HandleFunc("/api/login", handlers.Login).Methods("POST")

	// Document routes
	api.HandleFunc("/documents", documentService.UploadDocument).Methods("POST")
	api.HandleFunc("/documents", documentService.ListDocuments).Methods("GET")
	api.HandleFunc("/documents/{id}", documentService.GetDocument).Methods("GET")
	api.HandleFunc("/documents/{id}", documentService.DeleteDocument).Methods("DELETE")

	// Insight routes
	api.HandleFunc("/insights/{documentId}", llmService.GenerateInsight).Methods("POST")
	api.HandleFunc("/chat/{documentId}", llmService.ChatWithDocument).Methods("POST")
	api.HandleFunc("/chat/history/{documentId}", llmService.GetChatHistory).Methods("GET")

	// CORS configuration
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, c.Handler(r)))
}