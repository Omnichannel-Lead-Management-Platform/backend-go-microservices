package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/omnichannel/auth_service/internal/db"
)

// MockQuerier is a mock implementation of db.Querier
type MockQuerier struct {
	mock.Mock
}

func (m *MockQuerier) GetSessionByToken(ctx context.Context, token string) (db.Session, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(db.Session), args.Error(1)
}

func (m *MockQuerier) GetUserByID(ctx context.Context, id uuid.UUID) (db.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.User), args.Error(1)
}

func (m *MockQuerier) GetWorkspaceByID(ctx context.Context, id uuid.UUID) (db.Workspace, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.Workspace), args.Error(1)
}

func TestValidateSession_Success(t *testing.T) {
	mockDB := new(MockQuerier)
	authService := NewAuthService(mockDB)

	token := "valid_token"
	userID := uuid.New()
	workspaceID := uuid.New()

	session := db.Session{
		ID:        uuid.New(),
		UserId:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour), // Valid
	}

	user := db.User{
		ID:          userID,
		WorkspaceID: workspaceID,
		Role:        "admin",
	}

	mockDB.On("GetSessionByToken", mock.Anything, token).Return(session, nil)
	mockDB.On("GetUserByID", mock.Anything, userID).Return(user, nil)

	ctx := context.Background()
	result, err := authService.ValidateSession(ctx, token)

	assert.NoError(t, err)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, workspaceID, result.WorkspaceID)
	assert.Equal(t, "admin", result.Role)
	mockDB.AssertExpectations(t)
}

func TestValidateSession_Expired(t *testing.T) {
	mockDB := new(MockQuerier)
	authService := NewAuthService(mockDB)

	token := "expired_token"
	session := db.Session{
		Token:     token,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}

	mockDB.On("GetSessionByToken", mock.Anything, token).Return(session, nil)

	ctx := context.Background()
	_, err := authService.ValidateSession(ctx, token)

	assert.Error(t, err)
	assert.Equal(t, ErrSessionExpired, err)
	mockDB.AssertExpectations(t)
}

func TestValidateSession_NotFound(t *testing.T) {
	mockDB := new(MockQuerier)
	authService := NewAuthService(mockDB)

	token := "invalid_token"

	mockDB.On("GetSessionByToken", mock.Anything, token).Return(db.Session{}, errors.New("not found"))

	ctx := context.Background()
	_, err := authService.ValidateSession(ctx, token)

	assert.Error(t, err)
	assert.Equal(t, ErrSessionNotFound, err)
	mockDB.AssertExpectations(t)
}
