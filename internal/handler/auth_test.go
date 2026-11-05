package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthHandler_LoginSuccess(t *testing.T) {
	h := NewAuthHandler()

	body := bytes.NewBufferString(`{"username":"admin","password":"admin"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", body)
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var res map[string]string
	err := json.NewDecoder(rec.Body).Decode(&res)
	require.NoError(t, err)
	assert.NotEmpty(t, res["token"])
}

func TestAuthHandler_LoginUnauthorized(t *testing.T) {
	h := NewAuthHandler()

	body := bytes.NewBufferString(`{"username":"admin","password":"wrongpassword"}`)
	req := httptest.NewRequest(http.MethodPost, "/login", body)
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthHandler_LoginInvalidJSON(t *testing.T) {
	h := NewAuthHandler()

	body := bytes.NewBufferString(`{invalid json}`)
	req := httptest.NewRequest(http.MethodPost, "/login", body)
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
