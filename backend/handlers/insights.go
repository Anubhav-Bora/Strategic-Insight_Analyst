package handlers

import (
	"bytes"
	"context"
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
	return &LLMService{db: db}
}

type ChatMessage struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content"`
}

type ChatHistoryItem struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
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

func (ls *LLMService) getChunks(ctx context.Context, documentID string) ([]string, error) {
	rows, err := ls.db.QueryContext(ctx, `
		SELECT content FROM document_chunks
		WHERE document_id = $1
		ORDER BY chunk_index`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []string
	for rows.Next() {
		var chunk string
		if err := rows.Scan(&chunk); err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

func (ls *LLMService) getChatHistory(ctx context.Context, documentID, userID string) ([]ChatMessage, error) {
	rows, err := ls.db.QueryContext(ctx, `
		SELECT message_type, message_content
		FROM chat_history
		WHERE document_id = $1 AND user_id = $2
		ORDER BY timestamp
		LIMIT 10`, documentID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []ChatMessage
	for rows.Next() {
		var msgType, content string
		if err := rows.Scan(&msgType, &content); err != nil {
			return nil, err
		}
		role := "user"
		if msgType == "ai" {
			role = "model"
		}
		history = append(history, ChatMessage{Role: role, Content: content})
	}
	return history, nil
}

func (ls *LLMService) saveChat(ctx context.Context, documentID, userID, userMsg, aiMsg string) {
	_, err := ls.db.ExecContext(ctx, `
		INSERT INTO chat_history (id, document_id, user_id, message_type, message_content)
		VALUES ($1, $2, $3, 'user', $4),
		       ($5, $6, $7, 'ai', $8)`,
		uuid.New().String(), documentID, userID, userMsg,
		uuid.New().String(), documentID, userID, aiMsg)
	if err != nil {
		log.Printf("Warning: failed to save chat history: %v", err)
	}
}

func getUserID(ctx context.Context) (string, error) {
	userIDVal := ctx.Value(userIDKey)
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		return "", fmt.Errorf("userID missing in context")
	}
	return userID, nil
}

func (ls *LLMService) GenerateInsight(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := getUserID(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	documentID := mux.Vars(r)["documentId"]

	var req struct {
		Question string `json:"question"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	chunks, err := ls.getChunks(ctx, documentID)
	if err != nil || len(chunks) == 0 {
		http.Error(w, "Failed to retrieve document", http.StatusInternalServerError)
		return
	}
	contextText := selectRelevantChunks(chunks, req.Question, 2000)

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

Your Response:`, contextText, req.Question)

	response, err := ls.callHuggingFaceAPI(prompt)
	if err != nil {
		http.Error(w, "LLM API error", http.StatusInternalServerError)
		return
	}

	ls.saveChat(ctx, documentID, userID, req.Question, response)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"response": response})
}

func (ls *LLMService) ChatWithDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := getUserID(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	documentID := mux.Vars(r)["documentId"]

	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	chunks, err := ls.getChunks(ctx, documentID)
	if err != nil || len(chunks) == 0 {
		http.Error(w, "Failed to retrieve document", http.StatusInternalServerError)
		return
	}
	contextText := selectRelevantChunks(chunks, req.Message, 2000)

	history, err := ls.getChatHistory(ctx, documentID, userID)
	if err != nil {
		http.Error(w, "Chat history error", http.StatusInternalServerError)
		return
	}

	var prompt strings.Builder
	prompt.WriteString(`You are a Strategic Insight Analyst. I will provide you with a business document and you will help me analyze it.

Instructions:
- Provide a clear, concise, and well-structured response.
- Use bullet points or numbered lists for key points.
- Highlight strategic implications and actionable insights.
- If the answer is not in the document, state that clearly.
- Use simple language and avoid jargon.

Document Context:
`)
	prompt.WriteString(contextText + "\n\n")
	for _, msg := range history {
		prefix := "User: "
		if msg.Role == "model" {
			prefix = "AI: "
		}
		prompt.WriteString(prefix + msg.Content + "\n")
	}
	prompt.WriteString("User: " + req.Message + "\nAI:")

	response, err := ls.callHuggingFaceAPI(prompt.String())
	if err != nil {
		http.Error(w, "LLM API error", http.StatusInternalServerError)
		return
	}

	ls.saveChat(ctx, documentID, userID, req.Message, response)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"response": response})
}

func (ls *LLMService) GetChatHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, err := getUserID(ctx)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	documentID := mux.Vars(r)["documentId"]

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

	var history []ChatHistoryItem
	for rows.Next() {
		var item ChatHistoryItem
		if err := rows.Scan(&item.ID, &item.Type, &item.Content, &item.Timestamp); err != nil {
			http.Error(w, "Error scanning chat history", http.StatusInternalServerError)
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

	body, err := json.Marshal(map[string]interface{}{
		"inputs": prompt,
		"parameters": map[string]interface{}{
			"max_new_tokens":   256,
			"return_full_text": false,
		},
	})
	if err != nil {
		return "", fmt.Errorf("marshal error: %v", err)
	}

	req, err := http.NewRequest("POST", hfURL, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("request error: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API call error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("HuggingFace error %d: %s", resp.StatusCode, string(b))
	}

	var hfResp []struct {
		GeneratedText string `json:"generated_text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&hfResp); err != nil {
		return "", fmt.Errorf("decode error: %v", err)
	}
	if len(hfResp) == 0 {
		return "", fmt.Errorf("no content in response")
	}
	return hfResp[0].GeneratedText, nil
}
