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
	userID := ctx.Value("userID").(string)

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
	if _, err := os.Stat("uploads"); os.IsNotExist(err) {
		os.Mkdir("uploads", 0755)
	}
	uploadPath := filepath.Join("uploads", newFileName)

	out, err := os.Create(uploadPath)
	if err != nil {
		http.Error(w, "Unable to save file", http.StatusInternalServerError)
		return
	}
	defer out.Close()
	_, err = io.Copy(out, file)
	if err != nil {
		http.Error(w, "Unable to save file", http.StatusInternalServerError)
		return
	}

	// Extract text
	var textContent string
	if fileExt == ".pdf" {
		textContent, err = extractTextFromPDF(uploadPath)
		if err != nil {
			http.Error(w, "Failed to extract text", http.StatusInternalServerError)
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

	docID := uuid.New().String()
	_, err = ds.db.ExecContext(ctx, `
		INSERT INTO documents (id, user_id, file_name, storage_path, uploaded_at)
		VALUES ($1, $2, $3, $4, $5)`,
		docID, userID, handler.Filename, uploadPath, time.Now())
	if err != nil {
		log.Printf("Database error (saving document): %v", err)
		http.Error(w, "Error saving document to database", http.StatusInternalServerError)
		return
	}

	err = ds.saveDocumentChunks(ctx, docID, textContent)
	if err != nil {
		log.Printf("Database error (saving chunks): %v", err)
		http.Error(w, "Error saving document chunks", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"id":         docID,
		"fileName":   handler.Filename,
		"storageUrl": uploadPath,
		"uploadedAt": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ds *DocumentService) saveDocumentChunks(ctx context.Context, documentID string, content string) error {
	chunkSize := 2000
	length := len(content)
	numChunks := (length + chunkSize - 1) / chunkSize

	for i := 0; i < numChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > length {
			end = length
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
		p := f.Page(i)
		content := p.Content()
		for _, txt := range content.Text {
			text += txt.S + " "
		}
		text += "\n"
	}
	return text, nil
}

func (ds *DocumentService) ListDocuments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("userID").(string)

	rows, err := ds.db.QueryContext(ctx, `
		SELECT id, file_name, storage_path, uploaded_at
		FROM documents
		WHERE user_id = $1
		ORDER BY uploaded_at DESC`, userID)
	if err != nil {
		log.Printf("Database error (listing documents): %v", err)
		http.Error(w, "Error querying documents", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var documents []Document
	for rows.Next() {
		var doc Document
		err := rows.Scan(&doc.ID, &doc.FileName, &doc.StorageURL, &doc.UploadedAt)
		if err != nil {
			log.Printf("Database error (scanning document row): %v", err)
			http.Error(w, "Error scanning document row", http.StatusInternalServerError)
			return
		}
		doc.UserID = userID
		documents = append(documents, doc)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(documents)
}

func (ds *DocumentService) GetDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("userID").(string)
	vars := mux.Vars(r)
	docID := vars["id"]

	var doc Document
	err := ds.db.QueryRowContext(ctx, `
		SELECT id, user_id, file_name, storage_path, uploaded_at
		FROM documents
		WHERE id = $1 AND user_id = $2`, docID, userID).Scan(
		&doc.ID, &doc.UserID, &doc.FileName, &doc.StorageURL, &doc.UploadedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Document not found", http.StatusNotFound)
			return
		}
		log.Printf("Database error (retrieving document): %v", err)
		http.Error(w, "Error retrieving document", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func (ds *DocumentService) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("userID").(string)
	vars := mux.Vars(r)
	docID := vars["id"]

	var storagePath string
	err := ds.db.QueryRowContext(ctx, `
		SELECT storage_path FROM documents WHERE id = $1 AND user_id = $2`, docID, userID).Scan(&storagePath)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Document not found", http.StatusNotFound)
			return
		}
		log.Printf("Database error (retrieving document for deletion): %v", err)
		http.Error(w, "Error retrieving document for deletion", http.StatusInternalServerError)
		return
	}

	// Delete file from local storage
	_ = os.Remove(storagePath)

	_, err = ds.db.ExecContext(ctx, "DELETE FROM documents WHERE id = $1 AND user_id = $2", docID, userID)
	if err != nil {
		log.Printf("Database error (deleting document): %v", err)
		http.Error(w, "Error deleting document from database", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
