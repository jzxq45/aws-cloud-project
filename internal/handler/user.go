package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"aws-local-test/internal/middleware"
	"aws-local-test/internal/model"
	"aws-local-test/internal/repository"
)

type UserHandler struct {
	repo repository.UserRepository
}

func NewUserHandler(repo repository.UserRepository) *UserHandler {
	return &UserHandler{
		repo: repo,
	}
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var newUser model.User
	if err := json.NewDecoder(r.Body).Decode(&newUser); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if newUser.ID == "" || newUser.Name == "" || newUser.Email == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	if err := h.repo.Create(r.Context(), newUser); err != nil {
		log.Printf("Create user failed: %v", err)
		http.Error(w, "Failed to save to database", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "User created"}); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	user, err := h.repo.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		log.Printf("Get user failed: %v", err)
		http.Error(w, "Failed to fetch user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(user); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	var updatedUser model.User
	if err := json.NewDecoder(r.Body).Decode(&updatedUser); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if updatedUser.Name == "" || updatedUser.Email == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	updatedUser.ID = userID

	if err := h.repo.UpdateUser(r.Context(), updatedUser); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		log.Printf("Update user failed: %v", err)
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(updatedUser); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	if err := h.repo.DeleteUser(r.Context(), userID); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		log.Printf("Delete user failed: %v", err)
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "User deleted"}); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.repo.ListUsers(r.Context())
	if err != nil {
		log.Printf("List users failed: %v", err)
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(users); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func NewRouter(h *UserHandler, aiH *AIHandler, m *middleware.Metrics) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /users", h.CreateUser)
	mux.HandleFunc("GET /users/{id}", h.GetUser)
	mux.HandleFunc("PUT /users/{id}", h.UpdateUser)
	mux.HandleFunc("DELETE /users/{id}", h.DeleteUser)
	mux.HandleFunc("GET /users", h.ListUsers)

	if aiH != nil {
		mux.HandleFunc("POST /ask", aiH.Ask)
	}

	if m != nil {
		mux.Handle("GET /metrics", m.Handler())
		return m.Wrap(mux)
	}
	return mux
}
