package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/omnichannel/auth_service/internal/service"
)

// MockAuthService is a mock of the AuthService
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) ValidateSession(ctx context.Context, token string) (*service.ValidationResult, error) {
	args := m.Called(ctx, token)
	if args.Get(0) != nil {
		return args.Get(0).(*service.ValidationResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestValidateSessionHandler_Success(t *testing.T) {
	mockSvc := new(MockAuthService)
	handler := NewAuthHandler(mockSvc)
	router := chi.NewRouter()
	router.Post("/api/auth/validate", handler.ValidateSessionHandler)

	token := "valid_better_auth_token"
	userID := uuid.New()
	workspaceID := uuid.New()

	mockResult := &service.ValidationResult{
		UserID:      userID,
		WorkspaceID: workspaceID,
		Role:        "admin",
	}

	mockSvc.On("ValidateSession", mock.Anything, token).Return(mockResult, nil)

	req, _ := http.NewRequest("POST", "/api/auth/validate", nil)
	// Mocking BetterAuth cookie
	req.AddCookie(&http.Cookie{Name: "better-auth.session_token", Value: token})

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, userID.String(), response["user_id"])
	assert.Equal(t, workspaceID.String(), response["workspace_id"])
	assert.Equal(t, "admin", response["role"])

	mockSvc.AssertExpectations(t)
}

func TestValidateSessionHandler_MissingCookie(t *testing.T) {
	mockSvc := new(MockAuthService)
	handler := NewAuthHandler(mockSvc)
	router := chi.NewRouter()
	router.Post("/api/auth/validate", handler.ValidateSessionHandler)

	req, _ := http.NewRequest("POST", "/api/auth/validate", nil)
	// No cookie added

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestValidateSessionHandler_InvalidSession(t *testing.T) {
	mockSvc := new(MockAuthService)
	handler := NewAuthHandler(mockSvc)
	router := chi.NewRouter()
	router.Post("/api/auth/validate", handler.ValidateSessionHandler)

	token := "invalid_token"

	mockSvc.On("ValidateSession", mock.Anything, token).Return(nil, service.ErrSessionExpired)

	req, _ := http.NewRequest("POST", "/api/auth/validate", nil)
	req.AddCookie(&http.Cookie{Name: "better-auth.session_token", Value: token})

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	mockSvc.AssertExpectations(t)
}
