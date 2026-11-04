package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOllamaClient_Generate_Success(t *testing.T) {
	expectedPrompt := "Why Go?"
	expectedResponse := "Go is fast and simple."
	modelName := "qwen2.5:3b"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/generate", r.URL.Path)

		var reqBody map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)

		assert.Equal(t, modelName, reqBody["model"])
		assert.Equal(t, expectedPrompt, reqBody["prompt"])
		assert.Equal(t, false, reqBody["stream"])

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"model":    modelName,
			"response": expectedResponse,
			"done":     true,
		})
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, modelName, server.Client())
	resp, err := client.Generate(context.Background(), expectedPrompt)

	require.NoError(t, err)
	assert.Equal(t, expectedResponse, resp)
}

func TestOllamaClient_Generate_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal model error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "qwen2.5:3b", server.Client())
	resp, err := client.Generate(context.Background(), "test")

	require.Error(t, err)
	assert.Empty(t, resp)
	assert.Contains(t, err.Error(), "unexpected status code: 500")
}

func TestOllamaClient_Generate_BadJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{invalid-json"))
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "qwen2.5:3b", server.Client())
	resp, err := client.Generate(context.Background(), "test")

	require.Error(t, err)
	assert.Empty(t, resp)
	assert.Contains(t, err.Error(), "failed to decode ollama response")
}

func TestOllamaClient_Generate_ContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte(`{"response": "too late"}`))
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, "qwen2.5:3b", server.Client())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	resp, err := client.Generate(ctx, "test")

	require.Error(t, err)
	assert.Empty(t, resp)
}
