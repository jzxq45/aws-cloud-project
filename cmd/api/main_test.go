package main

import (
	"context"
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
	router := handler.NewRouter(userHandler, nil, middleware.NewMetrics())
	require.NotNil(t, router)

	// Test GET /users/{id}
	reqGet := httptest.NewRequest(http.MethodGet, "/users/test-1", nil)
	recGet := httptest.NewRecorder()
	router.ServeHTTP(recGet, reqGet)
	assert.Equal(t, http.StatusOK, recGet.Code)
	assert.Contains(t, recGet.Body.String(), "test@example.com")

	// Test GET /users
	reqList := httptest.NewRequest(http.MethodGet, "/users", nil)
	recList := httptest.NewRecorder()
	router.ServeHTTP(recList, reqList)
	assert.Equal(t, http.StatusOK, recList.Code)
	assert.Contains(t, recList.Body.String(), "t1@ex.com")

	// Test GET / with web.Files
	routerWithWeb := handler.NewRouter(userHandler, nil, middleware.NewMetrics(), web.Files)
	reqRoot := httptest.NewRequest(http.MethodGet, "/", nil)
	recRoot := httptest.NewRecorder()
	routerWithWeb.ServeHTTP(recRoot, reqRoot)
	assert.Equal(t, http.StatusOK, recRoot.Code)
	assert.Contains(t, recRoot.Body.String(), "AWS Cloud Lab")
}
