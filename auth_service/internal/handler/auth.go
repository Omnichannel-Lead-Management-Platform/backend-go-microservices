package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/omnichannel/auth_service/internal/service"
)

// SessionValidator defines the interface for session validation (satisfied by AuthService)
type SessionValidator interface {
	ValidateSession(ctx context.Context, token string) (*service.ValidationResult, error)
}

type AuthHandler struct {
	authService SessionValidator
}

func NewAuthHandler(authService SessionValidator) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/api/auth/validate", h.ValidateSessionHandler)
}

func (h *AuthHandler) ValidateSessionHandler(w http.ResponseWriter, r *http.Request) {
	// Better Auth sets a cookie named "better-auth.session_token" by default
	cookie, err := r.Cookie("better-auth.session_token")
	if err != nil {
		http.Error(w, "Unauthorized: missing session token", http.StatusUnauthorized)
		return
	}

	token := cookie.Value

	result, err := h.authService.ValidateSession(r.Context(), token)
	if err != nil {
		http.Error(w, "Unauthorized: invalid or expired session", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"user_id":      result.UserID.String(),
		"workspace_id": result.WorkspaceID.String(),
		"role":         result.Role,
	})
}
