package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"strategic-insight-analyst/utils"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	storage "github.com/supabase-community/storage-go"
	"rsc.io/pdf"
)

var (
	supabaseURL    = utils.GetEnv("SUPABASE_URL", "")
	supabaseKey    = utils.GetEnv("SUPABASE_SERVICE_ROLE_KEY", "")
	supabaseBucket = utils.GetEnv("SUPABASE_BUCKET", "")
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
	log.Printf("UploadDocument: userID=%s", userID)

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

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		http.Error(w, "Unable to read file", http.StatusInternalServerError)
		return
	}

	client := storage.NewClient(supabaseURL+"/storage/v1", supabaseKey, nil)
	uploadPath := "documents/" + newFileName

	_, err = client.UploadFile(supabaseBucket, uploadPath, &buf)
	if err != nil {
		log.Printf("Supabase upload error: %v", err)
		http.Error(w, "Failed to upload to Supabase Storage", http.StatusInternalServerError)
		return
	}

	downloaded, err := client.DownloadFile(supabaseBucket, uploadPath)
	if err != nil {
		log.Printf("Supabase download error: %v", err)
		http.Error(w, "Failed to download file from Supabase Storage", http.StatusInternalServerError)
		return
	}

	var textContent string
	if fileExt == ".pdf" {

		tmpFile, err := os.CreateTemp("", "*.pdf")
		if err != nil {
			http.Error(w, "Failed to create temp file", http.StatusInternalServerError)
			return
		}
		defer os.Remove(tmpFile.Name())
		if _, err := tmpFile.Write(downloaded); err != nil {
			http.Error(w, "Failed to write temp file", http.StatusInternalServerError)
			return
		}
		tmpFile.Close()
		textContent, err = extractTextFromPDF(tmpFile.Name())
		if err != nil {
			http.Error(w, "Failed to extract text from PDF", http.StatusInternalServerError)
			return
		}
	} else if fileExt == ".txt" {

		textContent = string(downloaded)
	}
	textContent = strings.ReplaceAll(textContent, "\x00", "")

	docID := uuid.New().String()
	uploadedAt := time.Now()

	garbled := false
	asciiCount := 0
	for _, r := range textContent {
		if r >= 32 && r <= 126 {
			asciiCount++
		}
	}
	if len(textContent) > 0 && (asciiCount < len(textContent)/2) {
		garbled = true
	}
	if garbled {
		log.Printf("[WARNING] Extracted text may be garbled for docID=%s, fileName=%s", docID, handler.Filename)
	}

	_, err = ds.db.ExecContext(ctx, `
		INSERT INTO documents (id, user_id, file_name, storage_path, uploaded_at)
		VALUES ($1, $2, $3, $4, $5)`,
		docID, userID, handler.Filename, uploadPath, uploadedAt)
	if err != nil {
		log.Printf("Database error (insert document): %v", err)
		http.Error(w, "Error saving document to database", http.StatusInternalServerError)
		return
	}

	if textContent != "" {
		if err := ds.saveDocumentChunks(ctx, docID, textContent); err != nil {
			log.Printf("Database error (saving chunks): %v", err)
			http.Error(w, "Error saving document chunks", http.StatusInternalServerError)
			return
		}
	}

	publicUrl := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", supabaseURL, supabaseBucket, uploadPath)

	response := Document{
		ID:         docID,
		UserID:     userID,
		FileName:   handler.Filename,
		StorageURL: publicUrl,
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

	apiKey := os.Getenv("OCR_SPACE_API_KEY")
	if apiKey != "" {
		ocrText, ocrErr := extractTextWithOCRSpace(filePath, apiKey)
		if ocrErr == nil && len(ocrText) > 0 {
			return ocrText, nil
		}
	}

	txtPath := filePath + ".txt"
	cmd := exec.Command("pdftotext", filePath, txtPath)
	err := cmd.Run()
	if err == nil {
		defer os.Remove(txtPath)
		data, readErr := ioutil.ReadFile(txtPath)
		if readErr == nil && len(data) > 0 {
			return string(data), nil
		}
	}

	f, err := pdf.Open(filePath)
	if err == nil {
		var text string
		for i := 1; i <= f.NumPage(); i++ {
			page := f.Page(i)
			content := page.Content()
			for _, txt := range content.Text {
				text += txt.S + " "
			}
			text += "\n"
		}
		if len(text) > 20 {
			return text, nil
		}
	}
	return "", fmt.Errorf("Failed to extract text from PDF")
}

func extractTextWithOCRSpace(pdfPath, apiKey string) (string, error) {
	file, err := os.Open(pdfPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("file", pdfPath)
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(fw, file); err != nil {
		return "", err
	}
	w.WriteField("language", "eng")
	w.WriteField("isOverlayRequired", "false")
	w.WriteField("OCREngine", "2")
	w.Close()

	req, err := http.NewRequest("POST", "https://api.ocr.space/parse/image", &b)
	if err != nil {
		return "", err
	}
	req.Header.Set("apikey", apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ParsedResults []struct {
			ParsedText string `json:"ParsedText"`
		} `json:"ParsedResults"`
		IsErroredOnProcessing bool   `json:"IsErroredOnProcessing"`
		ErrorMessage          string `json:"ErrorMessage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.IsErroredOnProcessing {
		return "", fmt.Errorf("OCR.space error: %s", result.ErrorMessage)
	}
	if len(result.ParsedResults) == 0 {
		return "", fmt.Errorf("No text extracted")
	}
	return result.ParsedResults[0].ParsedText, nil
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
	log.Printf("GetDocument: docID=%s, userID=%s", docID, userID)

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
	log.Printf("DeleteDocument: docID=%s, userID=%s", docID, userID)

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

	client := storage.NewClient(supabaseURL+"/storage/v1", supabaseKey, nil)
	_, err = client.RemoveFile(supabaseBucket, []string{storagePath})
	if err != nil {
		log.Printf("Supabase delete error: %v", err)
		// Optionally, handle error but still delete DB record
	}

	_, err = ds.db.ExecContext(ctx, "DELETE FROM documents WHERE id = $1 AND user_id = $2", docID, userID)
	if err != nil {
		log.Printf("Database error (delete): %v", err)
		http.Error(w, "Failed to delete document", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
