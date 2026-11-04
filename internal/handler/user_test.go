package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"aws-local-test/internal/middleware"
	"aws-local-test/internal/model"
	"aws-local-test/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockUserRepository struct {
	CreateFunc     func(ctx context.Context, user model.User) error
	GetByIDFunc    func(ctx context.Context, id string) (*model.User, error)
	UpdateUserFunc func(ctx context.Context, user model.User) error
	DeleteUserFunc func(ctx context.Context, id string) error
	ListUsersFunc  func(ctx context.Context) ([]model.User, error)
}

func (m *MockUserRepository) Create(ctx context.Context, user model.User) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, user)
	}
	return nil
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockUserRepository) UpdateUser(ctx context.Context, user model.User) error {
	if m.UpdateUserFunc != nil {
		return m.UpdateUserFunc(ctx, user)
	}
	return nil
}

func (m *MockUserRepository) DeleteUser(ctx context.Context, id string) error {
	if m.DeleteUserFunc != nil {
		return m.DeleteUserFunc(ctx, id)
	}
	return nil
}

func (m *MockUserRepository) ListUsers(ctx context.Context) ([]model.User, error) {
	if m.ListUsersFunc != nil {
		return m.ListUsersFunc(ctx)
	}
	return []model.User{}, nil
}

func TestUserHandler_CreateUser_MalformedJSON(t *testing.T) {
	h := NewUserHandler(&MockUserRepository{})
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString("{bad-json"))
	rec := httptest.NewRecorder()

	h.CreateUser(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid JSON")
}

func TestUserHandler_CreateUser_MissingFields(t *testing.T) {
	h := NewUserHandler(&MockUserRepository{})
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(`{"id":"","name":"A","email":""}`))
	rec := httptest.NewRecorder()

	h.CreateUser(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Missing required fields")
}

func TestUserHandler_CreateUser_RepoError(t *testing.T) {
	mockRepo := &MockUserRepository{
		CreateFunc: func(ctx context.Context, user model.User) error {
			return errors.New("database down")
		},
	}
	h := NewUserHandler(mockRepo)
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(`{"id":"u1","name":"A","email":"a@b.com"}`))
	rec := httptest.NewRecorder()

	h.CreateUser(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to save to database")
}

func TestUserHandler_CreateUser_Success(t *testing.T) {
	called := false
	mockRepo := &MockUserRepository{
		CreateFunc: func(ctx context.Context, user model.User) error {
			called = true
			assert.Equal(t, "u1", user.ID)
			return nil
		},
	}
	h := NewUserHandler(mockRepo)
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(`{"id":"u1","name":"A","email":"a@b.com"}`))
	rec := httptest.NewRecorder()

	h.CreateUser(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.True(t, called)
}

func TestUserHandler_GetUser_EmptyID(t *testing.T) {
	h := NewUserHandler(&MockUserRepository{})
	req := httptest.NewRequest(http.MethodGet, "/users/", nil)
	rec := httptest.NewRecorder()

	h.GetUser(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "User ID is required")
}

func TestUserHandler_GetUser_NotFound(t *testing.T) {
	mockRepo := &MockUserRepository{
		GetByIDFunc: func(ctx context.Context, id string) (*model.User, error) {
			return nil, repository.ErrUserNotFound
		},
	}
	h := NewUserHandler(mockRepo)
	req := httptest.NewRequest(http.MethodGet, "/users/unknown", nil)
	req.SetPathValue("id", "unknown")
	rec := httptest.NewRecorder()

	h.GetUser(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "User not found")
}

func TestUserHandler_GetUser_Success(t *testing.T) {
	expected := &model.User{
		ID:    "u42",
		Name:  "Grace",
		Email: "grace@example.com",
	}
	mockRepo := &MockUserRepository{
		GetByIDFunc: func(ctx context.Context, id string) (*model.User, error) {
			return expected, nil
		},
	}
	h := NewUserHandler(mockRepo)
	req := httptest.NewRequest(http.MethodGet, "/users/u42", nil)
	req.SetPathValue("id", "u42")
	rec := httptest.NewRecorder()

	h.GetUser(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var user model.User
	err := json.Unmarshal(rec.Body.Bytes(), &user)
	require.NoError(t, err)
	assert.Equal(t, *expected, user)
}

func TestUserHandler_UpdateUser_EmptyID(t *testing.T) {
	h := NewUserHandler(&MockUserRepository{})
	req := httptest.NewRequest(http.MethodPut, "/users/", bytes.NewBufferString(`{"name":"A","email":"a@b.com"}`))
	rec := httptest.NewRecorder()

	h.UpdateUser(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "User ID is required")
}

func TestUserHandler_UpdateUser_MalformedJSON(t *testing.T) {
	h := NewUserHandler(&MockUserRepository{})
	req := httptest.NewRequest(http.MethodPut, "/users/u1", bytes.NewBufferString(`{invalid-json`))
	req.SetPathValue("id", "u1")
	rec := httptest.NewRecorder()

	h.UpdateUser(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Invalid JSON")
}

func TestUserHandler_UpdateUser_MissingFields(t *testing.T) {
	h := NewUserHandler(&MockUserRepository{})
	req := httptest.NewRequest(http.MethodPut, "/users/u1", bytes.NewBufferString(`{"name":""}`))
	req.SetPathValue("id", "u1")
	rec := httptest.NewRecorder()

	h.UpdateUser(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Missing required fields")
}

func TestUserHandler_UpdateUser_NotFound(t *testing.T) {
	mockRepo := &MockUserRepository{
		UpdateUserFunc: func(ctx context.Context, user model.User) error {
			return repository.ErrUserNotFound
		},
	}
	h := NewUserHandler(mockRepo)
	req := httptest.NewRequest(http.MethodPut, "/users/u-missing", bytes.NewBufferString(`{"name":"Updated","email":"up@ex.com"}`))
	req.SetPathValue("id", "u-missing")
	rec := httptest.NewRecorder()

	h.UpdateUser(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "User not found")
}

func TestUserHandler_UpdateUser_Success(t *testing.T) {
	var updatedUser model.User
	mockRepo := &MockUserRepository{
		UpdateUserFunc: func(ctx context.Context, user model.User) error {
			updatedUser = user
			return nil
		},
	}
	h := NewUserHandler(mockRepo)
	req := httptest.NewRequest(http.MethodPut, "/users/u-1", bytes.NewBufferString(`{"name":"New Name","email":"new@ex.com"}`))
	req.SetPathValue("id", "u-1")
	rec := httptest.NewRecorder()

	h.UpdateUser(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.Equal(t, "u-1", updatedUser.ID)
	assert.Equal(t, "New Name", updatedUser.Name)
	assert.Equal(t, "new@ex.com", updatedUser.Email)
}

func TestUserHandler_DeleteUser_EmptyID(t *testing.T) {
	h := NewUserHandler(&MockUserRepository{})
	req := httptest.NewRequest(http.MethodDelete, "/users/", nil)
	rec := httptest.NewRecorder()

	h.DeleteUser(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "User ID is required")
}

func TestUserHandler_DeleteUser_NotFound(t *testing.T) {
	mockRepo := &MockUserRepository{
		DeleteUserFunc: func(ctx context.Context, id string) error {
			return repository.ErrUserNotFound
		},
	}
	h := NewUserHandler(mockRepo)
	req := httptest.NewRequest(http.MethodDelete, "/users/u-missing", nil)
	req.SetPathValue("id", "u-missing")
	rec := httptest.NewRecorder()

	h.DeleteUser(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "User not found")
}

func TestUserHandler_DeleteUser_Success(t *testing.T) {
	deletedID := ""
	mockRepo := &MockUserRepository{
		DeleteUserFunc: func(ctx context.Context, id string) error {
			deletedID = id
			return nil
		},
	}
	h := NewUserHandler(mockRepo)
	req := httptest.NewRequest(http.MethodDelete, "/users/u-delete", nil)
	req.SetPathValue("id", "u-delete")
	rec := httptest.NewRecorder()

	h.DeleteUser(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "u-delete", deletedID)
	assert.Contains(t, rec.Body.String(), "User deleted")
}

func TestUserHandler_ListUsers_Success(t *testing.T) {
	usersList := []model.User{
		{ID: "1", Name: "User 1", Email: "u1@test.com"},
		{ID: "2", Name: "User 2", Email: "u2@test.com"},
	}
	mockRepo := &MockUserRepository{
		ListUsersFunc: func(ctx context.Context) ([]model.User, error) {
			return usersList, nil
		},
	}
	h := NewUserHandler(mockRepo)
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()

	h.ListUsers(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var res []model.User
	err := json.Unmarshal(rec.Body.Bytes(), &res)
	require.NoError(t, err)
	assert.Len(t, res, 2)
	assert.Equal(t, usersList, res)
}

func TestUserHandler_ListUsers_RepoError(t *testing.T) {
	mockRepo := &MockUserRepository{
		ListUsersFunc: func(ctx context.Context) ([]model.User, error) {
			return nil, errors.New("dynamo scan failed")
		},
	}
	h := NewUserHandler(mockRepo)
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()

	h.ListUsers(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to fetch users")
}

func TestNewRouter(t *testing.T) {
	mockRepo := &MockUserRepository{
		ListUsersFunc: func(ctx context.Context) ([]model.User, error) {
			return []model.User{}, nil
		},
	}
	h := NewUserHandler(mockRepo)
	aiH := NewAIHandler(&MockLLMClient{
		GenerateFunc: func(ctx context.Context, prompt string) (string, error) {
			return "AI answer", nil
		},
	})
	m := middleware.NewMetrics()
	router := NewRouter(h, aiH, m)
	require.NotNil(t, router)

	// Test GET /users
	reqList := httptest.NewRequest(http.MethodGet, "/users", nil)
	recList := httptest.NewRecorder()
	router.ServeHTTP(recList, reqList)
	assert.Equal(t, http.StatusOK, recList.Code)

	// Test GET /metrics
	reqMetrics := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	recMetrics := httptest.NewRecorder()
	router.ServeHTTP(recMetrics, reqMetrics)
	assert.Equal(t, http.StatusOK, recMetrics.Code)
	assert.Contains(t, recMetrics.Body.String(), `http_requests_total{method="GET",path="/users",status="200"} 1`)

	// Test POST /ask
	reqAsk := httptest.NewRequest(http.MethodPost, "/ask", bytes.NewBufferString(`{"prompt": "Hello AI"}`))
	recAsk := httptest.NewRecorder()
	router.ServeHTTP(recAsk, reqAsk)
	assert.Equal(t, http.StatusOK, recAsk.Code)
	assert.Contains(t, recAsk.Body.String(), "AI answer")

	// Test 404
	req := httptest.NewRequest(http.MethodGet, "/unknown-route", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
