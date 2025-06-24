package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type LLMService struct {
	apiKey string
	db     *sql.DB
}

func NewLLMService(db *sql.DB) *LLMService {
	return &LLMService{
		apiKey: os.Getenv("GEMINI_API_KEY"),
		db:     db,
	}
}

type ChatMessage struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content"`
}

type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

type GeminiContent struct {
	Role  string         `json:"role,omitempty"`
	Parts []GeminiPart   `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (ls *LLMService) GenerateInsight(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("userID").(string)
	vars := mux.Vars(r)
	documentID := vars["documentId"]

	var request struct {
		Question string `json:"question"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get document chunks from database
	rows, err := ls.db.QueryContext(ctx, `
		SELECT content FROM document_chunks
		WHERE document_id = $1
		ORDER BY chunk_index`, documentID)
	if err != nil {
		http.Error(w, "Error retrieving document chunks", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var chunks []string
	for rows.Next() {
		var chunk string
		if err := rows.Scan(&chunk); err != nil {
			http.Error(w, "Error scanning document chunk", http.StatusInternalServerError)
			return
		}
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		http.Error(w, "No document content found", http.StatusNotFound)
		return
	}

	// Combine chunks into context (limit to first few chunks to avoid hitting token limits)
	context := strings.Join(chunks[:min(5, len(chunks))], "\n\n")

	// Construct prompt
	prompt := fmt.Sprintf(`You are a Strategic Insight Analyst. Analyze the following business document and provide insights based on the user's question.

Document Context:
%s

User Question: %s

Instructions:
1. Provide a concise and analytical response.
2. Focus on strategic implications and key takeaways.
3. If the information is not available in the document, state that clearly.
4. Use bullet points for clarity when appropriate.`, context, request.Question)

	// Call Gemini API
	response, err := ls.callGeminiAPI(prompt)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error calling LLM API: %v", err), http.StatusInternalServerError)
		return
	}

	// Save to chat history
	chatID := uuid.New().String()
	_, err = ls.db.ExecContext(ctx, `
		INSERT INTO chat_history (id, document_id, user_id, message_type, message_content)
		VALUES ($1, $2, $3, 'user', $4),
		       ($5, $6, $7, 'ai', $8)`,
		uuid.New().String(), documentID, userID, request.Question,
		chatID, documentID, userID, response)
	if err != nil {
		log.Printf("Warning: failed to save chat history: %v", err)
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"response": response,
	})
}

func (ls *LLMService) ChatWithDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("userID").(string)
	vars := mux.Vars(r)
	documentID := vars["documentId"]

	var request struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get document chunks from database
	rows, err := ls.db.QueryContext(ctx, `
		SELECT content FROM document_chunks
		WHERE document_id = $1
		ORDER BY chunk_index`, documentID)
	if err != nil {
		http.Error(w, "Error retrieving document chunks", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var chunks []string
	for rows.Next() {
		var chunk string
		if err := rows.Scan(&chunk); err != nil {
			http.Error(w, "Error scanning document chunk", http.StatusInternalServerError)
			return
		}
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		http.Error(w, "No document content found", http.StatusNotFound)
		return
	}

	// Get previous chat history for context
	historyRows, err := ls.db.QueryContext(ctx, `
		SELECT message_type, message_content
		FROM chat_history
		WHERE document_id = $1 AND user_id = $2
		ORDER BY timestamp
		LIMIT 10`, documentID, userID)
	if err != nil {
		http.Error(w, "Error retrieving chat history", http.StatusInternalServerError)
		return
	}
	defer historyRows.Close()

	var chatHistory []ChatMessage
	for historyRows.Next() {
		var msgType, content string
		if err := historyRows.Scan(&msgType, &content); err != nil {
			http.Error(w, "Error scanning chat history", http.StatusInternalServerError)
			return
		}
		role := "user"
		if msgType == "ai" {
			role = "model"
		}
		chatHistory = append(chatHistory, ChatMessage{Role: role, Content: content})
	}

	// Combine chunks into context (limit to first few chunks to avoid hitting token limits)
	context := strings.Join(chunks[:min(5, len(chunks))], "\n\n")

	// Construct messages for Gemini
	messages := []ChatMessage{
		{
			Role: "user",
			Content: fmt.Sprintf(`You are a Strategic Insight Analyst. I will provide you with a business document and you will help me analyze it.

Document Context:
%s

Please answer my questions about this document with concise, analytical responses focusing on strategic implications.`, context),
		},
		{Role: "model", Content: "Understood. I'll analyze the document and provide strategic insights based on your questions."},
	}

	// Add chat history
	messages = append(messages, chatHistory...)

	// Add current message
	messages = append(messages, ChatMessage{Role: "user", Content: request.Message})

	// Call Gemini API
	response, err := ls.callGeminiChatAPI(messages)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error calling LLM API: %v", err), http.StatusInternalServerError)
		return
	}

	// Save to chat history
	_, err = ls.db.ExecContext(ctx, `
		INSERT INTO chat_history (id, document_id, user_id, message_type, message_content)
		VALUES ($1, $2, $3, 'user', $4),
		       ($5, $6, $7, 'ai', $8)`,
		uuid.New().String(), documentID, userID, request.Message,
		uuid.New().String(), documentID, userID, response)
	if err != nil {
		log.Printf("Warning: failed to save chat history: %v", err)
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"response": response,
	})
}

func (ls *LLMService) GetChatHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("userID").(string)
	vars := mux.Vars(r)
	documentID := vars["documentId"]

	rows, err := ls.db.QueryContext(ctx, `
		SELECT id, message_type, message_content, timestamp
		FROM chat_history
		WHERE document_id = $1 AND user_id = $2
		ORDER BY timestamp`, documentID, userID)
	if err != nil {
		http.Error(w, "Error retrieving chat history", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type ChatHistoryItem struct {
		ID        string    `json:"id"`
		Type      string    `json:"type"`
		Content   string    `json:"content"`
		Timestamp time.Time `json:"timestamp"`
	}

	var history []ChatHistoryItem
	for rows.Next() {
		var item ChatHistoryItem
		if err := rows.Scan(&item.ID, &item.Type, &item.Content, &item.Timestamp); err != nil {
			http.Error(w, "Error scanning chat history item", http.StatusInternalServerError)
			return
		}
		history = append(history, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func (ls *LLMService) callGeminiAPI(prompt string) (string, error) {
	geminiURL := "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent?key=" + ls.apiKey

	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request body: %v", err)
	}

	resp, err := http.Post(geminiURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("error making request to Gemini API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gemini API returned status %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", fmt.Errorf("error decoding Gemini response: %v", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content in Gemini response")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

func (ls *LLMService) callGeminiChatAPI(messages []ChatMessage) (string, error) {
	geminiURL := "https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent?key=" + ls.apiKey

	// Convert messages to Gemini format
	var contents []GeminiContent

	for _, msg := range messages {
		contents = append(contents, GeminiContent{
			Role: msg.Role,
			Parts: []GeminiPart{
				{Text: msg.Content},
			},
		})
	}

	reqBody := GeminiRequest{
		Contents: contents,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request body: %v", err)
	}

	resp, err := http.Post(geminiURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("error making request to Gemini API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gemini API returned status %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", fmt.Errorf("error decoding Gemini response: %v", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content in Gemini response")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}