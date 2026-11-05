package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aws-local-test/internal/handler"
	"aws-local-test/internal/middleware"
	"aws-local-test/internal/model"
	"aws-local-test/web"

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

func TestServerRouter_Wiring(t *testing.T) {
	mockRepo := &MockUserRepository{
		GetByIDFunc: func(ctx context.Context, id string) (*model.User, error) {
			return &model.User{ID: id, Name: "Tester", Email: "test@example.com"}, nil
		},
		ListUsersFunc: func(ctx context.Context) ([]model.User, error) {
			return []model.User{{ID: "1", Name: "T1", Email: "t1@ex.com"}}, nil
		},
	}

	userHandler := handler.NewUserHandler(mockRepo)
	authHandler := handler.NewAuthHandler()
	authMiddleware := middleware.NewAuthMiddleware()
	metrics := middleware.NewMetrics()

	router := handler.NewRouter(userHandler, nil, authHandler, metrics, authMiddleware, web.Files)
	require.NotNil(t, router)

	// Test GET / without auth (public static file)
	reqRoot := httptest.NewRequest(http.MethodGet, "/", nil)
	recRoot := httptest.NewRecorder()
	router.ServeHTTP(recRoot, reqRoot)
	assert.Equal(t, http.StatusOK, recRoot.Code)
	assert.Contains(t, recRoot.Body.String(), "AWS Cloud Lab")

	// Test GET /users without token -> 401 Unauthorized
	reqListUnauth := httptest.NewRequest(http.MethodGet, "/users", nil)
	recListUnauth := httptest.NewRecorder()
	router.ServeHTTP(recListUnauth, reqListUnauth)
	assert.Equal(t, http.StatusUnauthorized, recListUnauth.Code)

	// Test POST /login -> 200 OK with token
	loginBody := bytes.NewBufferString(`{"username":"admin","password":"admin"}`)
	reqLogin := httptest.NewRequest(http.MethodPost, "/login", loginBody)
	recLogin := httptest.NewRecorder()
	router.ServeHTTP(recLogin, reqLogin)
	assert.Equal(t, http.StatusOK, recLogin.Code)

	var loginResp map[string]string
	err := json.NewDecoder(recLogin.Body).Decode(&loginResp)
	require.NoError(t, err)
	token := loginResp["token"]
	assert.NotEmpty(t, token)

	// Test GET /users with valid Bearer token -> 200 OK
	reqList := httptest.NewRequest(http.MethodGet, "/users", nil)
	reqList.Header.Set("Authorization", "Bearer "+token)
	recList := httptest.NewRecorder()
	router.ServeHTTP(recList, reqList)
	assert.Equal(t, http.StatusOK, recList.Code)
	assert.Contains(t, recList.Body.String(), "t1@ex.com")

	// Test GET /users/{id} with token -> 200 OK
	reqGet := httptest.NewRequest(http.MethodGet, "/users/test-1", nil)
	reqGet.Header.Set("Authorization", "Bearer "+token)
	recGet := httptest.NewRecorder()
	router.ServeHTTP(recGet, reqGet)
	assert.Equal(t, http.StatusOK, recGet.Code)
	assert.Contains(t, recGet.Body.String(), "test@example.com")
}
