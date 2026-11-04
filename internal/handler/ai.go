package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"aws-local-test/internal/ai"
)

type AIHandler struct {
	client ai.LLMClient
}

func NewAIHandler(client ai.LLMClient) *AIHandler {
	return &AIHandler{
		client: client,
	}
}

type askRequest struct {
	Prompt string `json:"prompt"`
}

type askResponse struct {
	Response string `json:"response"`
}

func (h *AIHandler) Ask(w http.ResponseWriter, r *http.Request) {
	var req askRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		http.Error(w, "Prompt is required", http.StatusBadRequest)
		return
	}

	answer, err := h.client.Generate(r.Context(), prompt)
	if err != nil {
		log.Printf("LLM Generate error: %v", err)
		http.Error(w, "Failed to generate response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(askResponse{Response: answer}); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}
