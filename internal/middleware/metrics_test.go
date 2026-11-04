package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetrics_Middleware_RecordsRequestsAndDuration(t *testing.T) {
	m := NewMetrics()
	require.NotNil(t, m)

	handler := m.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad request"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	// 1. Send GET /success
	reqSuccess := httptest.NewRequest(http.MethodGet, "/success", nil)
	recSuccess := httptest.NewRecorder()
	handler.ServeHTTP(recSuccess, reqSuccess)
	assert.Equal(t, http.StatusOK, recSuccess.Code)

	// 2. Send POST /fail
	reqFail := httptest.NewRequest(http.MethodPost, "/fail", nil)
	recFail := httptest.NewRecorder()
	handler.ServeHTTP(recFail, reqFail)
	assert.Equal(t, http.StatusBadRequest, recFail.Code)

	// 3. Inspect /metrics output
	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	m.Handler().ServeHTTP(metricsRec, metricsReq)

	assert.Equal(t, http.StatusOK, metricsRec.Code)
	body := metricsRec.Body.String()

	// Verify counter metrics
	assert.Contains(t, body, `http_requests_total{method="GET",path="/success",status="200"} 1`)
	assert.Contains(t, body, `http_requests_total{method="POST",path="/fail",status="400"} 1`)

	// Verify histogram metrics
	assert.Contains(t, body, `http_request_duration_seconds_count{method="GET",path="/success"} 1`)
	assert.Contains(t, body, `http_request_duration_seconds_count{method="POST",path="/fail"} 1`)
}

func TestMetrics_Middleware_UsesPatternWhenAvailable(t *testing.T) {
	m := NewMetrics()
	require.NotNil(t, m)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("user"))
	})

	handler := m.Wrap(mux)

	// Send request with dynamic ID
	req := httptest.NewRequest(http.MethodGet, "/users/12345", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Inspect metrics: label 'path' should contain pattern "GET /users/{id}" instead of raw "/users/12345"
	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	m.Handler().ServeHTTP(metricsRec, metricsReq)

	assert.Equal(t, http.StatusOK, metricsRec.Code)
	body := metricsRec.Body.String()
	assert.Contains(t, body, `http_requests_total{method="GET",path="/users/{id}",status="200"} 1`)
	assert.NotContains(t, body, `12345`)
}
