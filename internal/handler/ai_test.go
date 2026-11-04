package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockLLMClient struct {
	GenerateFunc func(ctx context.Context, prompt string) (string, error)
}

func (m *MockLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, prompt)
	}
	return "", nil
}

func TestAIHandler_Ask_MalformedJSON(t *testing.T) {
	h := NewAIHandler(&MockLLMClient{})
	req := httptest.NewRequest(http.MethodPost, "/ask", bytes.NewBufferString("{bad-json"))
	rec := httptest.NewRecorder()

	h.Ask(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid JSON")
}

func TestAIHandler_Ask_EmptyPrompt(t *testing.T) {
	h := NewAIHandler(&MockLLMClient{})
	req := httptest.NewRequest(http.MethodPost, "/ask", bytes.NewBufferString(`{"prompt": "   "}`))
	rec := httptest.NewRecorder()

	h.Ask(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Prompt is required")
}

func TestAIHandler_Ask_LLMError(t *testing.T) {
	mockLLM := &MockLLMClient{
		GenerateFunc: func(ctx context.Context, prompt string) (string, error) {
			return "", errors.New("ollama offline")
		},
	}
	h := NewAIHandler(mockLLM)
	req := httptest.NewRequest(http.MethodPost, "/ask", bytes.NewBufferString(`{"prompt": "Hello"}`))
	rec := httptest.NewRecorder()

	h.Ask(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to generate response")
}

func TestAIHandler_Ask_Success(t *testing.T) {
	expectedPrompt := "Explain Go"
	expectedAnswer := "Go is a compiled language by Google."

	mockLLM := &MockLLMClient{
		GenerateFunc: func(ctx context.Context, prompt string) (string, error) {
			assert.Equal(t, expectedPrompt, prompt)
			return expectedAnswer, nil
		},
	}
	h := NewAIHandler(mockLLM)
	req := httptest.NewRequest(http.MethodPost, "/ask", bytes.NewBufferString(`{"prompt": "Explain Go"}`))
	rec := httptest.NewRecorder()

	h.Ask(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, expectedAnswer, resp["response"])
}
