package handler

import (
	"encoding/json"
	"log/slog"
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
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
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
		slog.Error("LLM Generate error", "error", err, "prompt_len", len(prompt))
		http.Error(w, "Failed to generate response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(askResponse{Response: answer}); err != nil {
		slog.Error("Failed to encode response", "error", err)
	}
}
