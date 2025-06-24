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

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"rsc.io/pdf"
)

type DocumentService struct {
	db            *sql.DB
	storageClient *storage.Client
}

func NewDocumentService(db *sql.DB, storageClient *storage.Client) *DocumentService {
	return &DocumentService{
		db:            db,
		storageClient: storageClient,
	}
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

	// Parse multipart form (max 10MB file size)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	// Get file from form data
	file, handler, err := r.FormFile("document")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Generate unique filename
	ext := filepath.Ext(handler.Filename)
	newFileName := uuid.New().String() + ext

	// Upload to Google Cloud Storage
	bucketName := os.Getenv("GCS_BUCKET")
	bucket := ds.storageClient.Bucket(bucketName)
	obj := bucket.Object(newFileName)
	wc := obj.NewWriter(ctx)

	if _, err = io.Copy(wc, file); err != nil {
		http.Error(w, "Error uploading file to storage", http.StatusInternalServerError)
		return
	}

	if err := wc.Close(); err != nil {
		http.Error(w, "Error closing storage writer", http.StatusInternalServerError)
		return
	}

	// Set ACL to make the file publicly readable
	if err := obj.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		http.Error(w, "Error setting file ACL", http.StatusInternalServerError)
		return
	}

	// Get public URL
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		http.Error(w, "Error getting file attributes", http.StatusInternalServerError)
		return
	}

	// Extract text from document
	var textContent string
	if ext == ".pdf" {
		// Create temp file for PDF processing
		tempFilePath := filepath.Join(os.TempDir(), newFileName)
		tempFile, err := os.Create(tempFilePath)
		if err != nil {
			http.Error(w, "Error creating temp file", http.StatusInternalServerError)
			return
		}
		defer os.Remove(tempFilePath)

		// Reset file reader to beginning
		if _, err := file.Seek(0, 0); err != nil {
			http.Error(w, "Error resetting file reader", http.StatusInternalServerError)
			return
		}

		if _, err := io.Copy(tempFile, file); err != nil {
			http.Error(w, "Error writing to temp file", http.StatusInternalServerError)
			return
		}
		tempFile.Close()

		// Extract text from PDF
		textContent, err = extractTextFromPDF(tempFilePath)
		if err != nil {
			http.Error(w, "Error extracting text from PDF", http.StatusInternalServerError)
			return
		}
	} else if ext == ".txt" {
		// For text files, just read the content
		contentBytes, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "Error reading text file", http.StatusInternalServerError)
			return
		}
		textContent = string(contentBytes)
	}

	// Save document metadata to database
	docID := uuid.New().String()
	_, err = ds.db.ExecContext(ctx, `
		INSERT INTO documents (id, user_id, file_name, storage_path, uploaded_at)
		VALUES ($1, $2, $3, $4, $5)`,
		docID, userID, handler.Filename, newFileName, time.Now())
	if err != nil {
		http.Error(w, "Error saving document to database", http.StatusInternalServerError)
		return
	}

	// Chunk the text content and save to database
	err = ds.saveDocumentChunks(ctx, docID, textContent)
	if err != nil {
		http.Error(w, "Error saving document chunks", http.StatusInternalServerError)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"id":         docID,
		"fileName":   handler.Filename,
		"storageUrl": attrs.MediaLink,
		"uploadedAt": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ds *DocumentService) saveDocumentChunks(ctx context.Context, documentID string, content string) error {
	// Split content into chunks of ~2000 characters (adjust based on your needs)
	chunkSize := 2000
	length := len(content)
	numChunks := (length + chunkSize - 1) / chunkSize // Round up division

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
		http.Error(w, "Error querying documents", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var documents []Document
	for rows.Next() {
		var doc Document
		err := rows.Scan(&doc.ID, &doc.FileName, &doc.StorageURL, &doc.UploadedAt)
		if err != nil {
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

	// First get the storage path
	var storagePath string
	err := ds.db.QueryRowContext(ctx, `
		SELECT storage_path FROM documents WHERE id = $1 AND user_id = $2`, docID, userID).Scan(&storagePath)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Document not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error retrieving document for deletion", http.StatusInternalServerError)
		return
	}

	// Delete from storage
	bucketName := os.Getenv("GCS_BUCKET")
	bucket := ds.storageClient.Bucket(bucketName)
	obj := bucket.Object(storagePath)
	if err := obj.Delete(ctx); err != nil {
		log.Printf("Warning: failed to delete file from storage: %v", err)
		// Continue with database deletion even if storage deletion fails
	}

	// Delete from database (cascade will delete chunks and chat history)
	_, err = ds.db.ExecContext(ctx, "DELETE FROM documents WHERE id = $1 AND user_id = $2", docID, userID)
	if err != nil {
		http.Error(w, "Error deleting document from database", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}