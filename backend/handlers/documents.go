package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"rsc.io/pdf"
)

type DocumentService struct {
	db *sql.DB
}

func NewDocumentService(db *sql.DB) *DocumentService {
	return &DocumentService{db: db}
}

type Document struct {
	ID         string    `json:"id"`
	UserID     string    `json:"userId"`
	FileName   string    `json:"fileName"`
	StorageURL string    `json:"storageUrl"`
	UploadedAt time.Time `json:"uploadedAt"`
}

func (ds *DocumentService) UploadDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userIDVal := ctx.Value(userIDKey)
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: userID missing in context", http.StatusUnauthorized)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("document")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileExt := filepath.Ext(handler.Filename)
	newFileName := uuid.New().String() + fileExt

	// Ensure uploads directory exists
	if err := os.MkdirAll("uploads", 0755); err != nil {
		http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
		return
	}

	uploadPath := filepath.Join("uploads", newFileName)
	out, err := os.Create(uploadPath)
	if err != nil {
		http.Error(w, "Unable to save file", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	if _, err = io.Copy(out, file); err != nil {
		http.Error(w, "Unable to save file", http.StatusInternalServerError)
		return
	}

	var textContent string
	if fileExt == ".pdf" {
		textContent, err = extractTextFromPDF(uploadPath)
		if err != nil {
			http.Error(w, "Failed to extract text from PDF", http.StatusInternalServerError)
			return
		}
	} else if fileExt == ".txt" {
		contentBytes, err := os.ReadFile(uploadPath)
		if err != nil {
			http.Error(w, "Failed to read text file", http.StatusInternalServerError)
			return
		}
		textContent = string(contentBytes)
	}

	textContent = strings.ReplaceAll(textContent, "\x00", "")

	docID := uuid.New().String()
	uploadedAt := time.Now()

	_, err = ds.db.ExecContext(ctx, `
		INSERT INTO documents (id, user_id, file_name, storage_path, uploaded_at)
		VALUES ($1, $2, $3, $4, $5)`,
		docID, userID, handler.Filename, uploadPath, uploadedAt)
	if err != nil {
		log.Printf("Database error (insert document): %v", err)
		http.Error(w, "Error saving document to database", http.StatusInternalServerError)
		return
	}

	if err := ds.saveDocumentChunks(ctx, docID, textContent); err != nil {
		log.Printf("Database error (saving chunks): %v", err)
		http.Error(w, "Error saving document chunks", http.StatusInternalServerError)
		return
	}

	response := Document{
		ID:         docID,
		UserID:     userID,
		FileName:   handler.Filename,
		StorageURL: uploadPath,
		UploadedAt: uploadedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ds *DocumentService) saveDocumentChunks(ctx context.Context, documentID string, content string) error {
	chunkSize := 2000
	for i := 0; i*chunkSize < len(content); i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(content) {
			end = len(content)
		}

		chunk := content[start:end]
		chunkID := uuid.New().String()

		_, err := ds.db.ExecContext(ctx, `
			INSERT INTO document_chunks (id, document_id, chunk_index, content)
			VALUES ($1, $2, $3, $4)`,
			chunkID, documentID, i, chunk)
		if err != nil {
			return fmt.Errorf("error inserting chunk %d: %v", i, err)
		}
	}
	return nil
}

func extractTextFromPDF(filePath string) (string, error) {
	f, err := pdf.Open(filePath)
	if err != nil {
		return "", err
	}
	var text string
	for i := 1; i <= f.NumPage(); i++ {
		page := f.Page(i)
		content := page.Content()
		for _, txt := range content.Text {
			text += txt.S + " "
		}
		text += "\n"
	}
	return text, nil
}

func (ds *DocumentService) ListDocuments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userIDVal := ctx.Value(userIDKey)
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: userID missing in context", http.StatusUnauthorized)
		return
	}

	rows, err := ds.db.QueryContext(ctx, `
		SELECT id, file_name, storage_path, uploaded_at
		FROM documents
		WHERE user_id = $1
		ORDER BY uploaded_at DESC`, userID)
	if err != nil {
		log.Printf("Database error (list): %v", err)
		http.Error(w, "Failed to list documents", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var documents []Document
	for rows.Next() {
		var doc Document
		if err := rows.Scan(&doc.ID, &doc.FileName, &doc.StorageURL, &doc.UploadedAt); err != nil {
			log.Printf("Row scan error: %v", err)
			continue
		}
		doc.UserID = userID
		documents = append(documents, doc)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(documents)
}

func (ds *DocumentService) GetDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userIDVal := ctx.Value(userIDKey)
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: userID missing in context", http.StatusUnauthorized)
		return
	}

	docID := mux.Vars(r)["id"]
	var doc Document

	err := ds.db.QueryRowContext(ctx, `
		SELECT id, user_id, file_name, storage_path, uploaded_at
		FROM documents WHERE id = $1 AND user_id = $2`,
		docID, userID).Scan(&doc.ID, &doc.UserID, &doc.FileName, &doc.StorageURL, &doc.UploadedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Document not found", http.StatusNotFound)
		} else {
			log.Printf("Database error (get): %v", err)
			http.Error(w, "Failed to retrieve document", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func (ds *DocumentService) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userIDVal := ctx.Value(userIDKey)
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: userID missing in context", http.StatusUnauthorized)
		return
	}

	docID := mux.Vars(r)["id"]

	var storagePath string
	err := ds.db.QueryRowContext(ctx, `
		SELECT storage_path FROM documents WHERE id = $1 AND user_id = $2`,
		docID, userID).Scan(&storagePath)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Document not found", http.StatusNotFound)
		} else {
			log.Printf("Database error (get for delete): %v", err)
			http.Error(w, "Failed to fetch document", http.StatusInternalServerError)
		}
		return
	}

	_ = os.Remove(storagePath)

	_, err = ds.db.ExecContext(ctx, "DELETE FROM documents WHERE id = $1 AND user_id = $2", docID, userID)
	if err != nil {
		log.Printf("Database error (delete): %v", err)
		http.Error(w, "Failed to delete document", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
