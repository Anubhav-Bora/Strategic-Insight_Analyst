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
	db *sql.DB
}

func NewLLMService(db *sql.DB) *LLMService {
	return &LLMService{
		db: db,
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
	Role  string       `json:"role,omitempty"`
	Parts []GeminiPart `json:"parts"`
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

func selectRelevantChunks(chunks []string, question string, maxChars int) string {
	var selected []string
	total := 0
	qWords := strings.Fields(strings.ToLower(question))
	for _, chunk := range chunks {
		lowerChunk := strings.ToLower(chunk)
		for _, word := range qWords {
			if strings.Contains(lowerChunk, word) && total+len(chunk) <= maxChars {
				selected = append(selected, chunk)
				total += len(chunk)
				break
			}
		}
		if total >= maxChars {
			break
		}
	}
	// Fallback: if nothing matched, use the first chunk(s)
	if len(selected) == 0 && len(chunks) > 0 {
		total = 0
		for _, chunk := range chunks {
			if total+len(chunk) > maxChars {
				break
			}
			selected = append(selected, chunk)
			total += len(chunk)
		}
	}
	return strings.Join(selected, "\n\n")
}

func (ls *LLMService) GenerateInsight(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userIDVal := ctx.Value(userIDKey)
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: userID missing in context", http.StatusUnauthorized)
		return
	}
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

	// Keyword-based chunk selection
	context := selectRelevantChunks(chunks, request.Question, 2000)

	// Construct prompt
	prompt := fmt.Sprintf(`You are a Strategic Insight Analyst. Analyze the following business document and answer the user's question.

Instructions:
- Provide a clear, concise, and well-structured response.
- Use bullet points or numbered lists for key points.
- Highlight strategic implications and actionable insights.
- If the answer is not in the document, state that clearly.
- Use simple language and avoid jargon.

Document Context:
%s

User Question:
%s

Your Response:`, context, request.Question)

	// Call Hugging Face API
	response, err := ls.callHuggingFaceAPI(prompt)
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
	userIDVal := ctx.Value(userIDKey)
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: userID missing in context", http.StatusUnauthorized)
		return
	}
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

	// Keyword-based chunk selection
	context := selectRelevantChunks(chunks, request.Message, 2000)

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

	// Build chat prompt: context + chat history + user message
	var chatPrompt strings.Builder
	chatPrompt.WriteString(`You are a Strategic Insight Analyst. I will provide you with a business document and you will help me analyze it.

Instructions:
- Provide a clear, concise, and well-structured response.
- Use bullet points or numbered lists for key points.
- Highlight strategic implications and actionable insights.
- If the answer is not in the document, state that clearly.
- Use simple language and avoid jargon.

Document Context:
`)
	chatPrompt.WriteString(context)
	chatPrompt.WriteString("\n\n")
	for _, msg := range chatHistory {
		if msg.Role == "user" {
			chatPrompt.WriteString("User: ")
		} else {
			chatPrompt.WriteString("AI: ")
		}
		chatPrompt.WriteString(msg.Content)
		chatPrompt.WriteString("\n")
	}
	chatPrompt.WriteString("User: ")
	chatPrompt.WriteString(request.Message)
	chatPrompt.WriteString("\nAI:")
	response, err := ls.callHuggingFaceAPI(chatPrompt.String())
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
	userIDVal := ctx.Value(userIDKey)
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized: userID missing in context", http.StatusUnauthorized)
		return
	}
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

func (ls *LLMService) callHuggingFaceAPI(prompt string) (string, error) {
	hfURL := "https://api-inference.huggingface.co/models/HuggingFaceH4/zephyr-7b-beta"
	apiKey := os.Getenv("HF_API_TOKEN")
	requestBody := map[string]interface{}{
		"inputs": prompt,
		"parameters": map[string]interface{}{
			"max_new_tokens":   256,
			"return_full_text": false,
		},
	}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling request body: %v", err)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("POST", hfURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request to Hugging Face API: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Hugging Face API returned status %d: %s", resp.StatusCode, string(body))
	}
	var hfResp []struct {
		GeneratedText string `json:"generated_text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&hfResp); err != nil {
		return "", fmt.Errorf("error decoding Hugging Face response: %v", err)
	}
	if len(hfResp) == 0 {
		return "", fmt.Errorf("no content in Hugging Face response")
	}
	return hfResp[0].GeneratedText, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
