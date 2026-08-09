package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/omnichannel/auth_service/internal/db"
)

var (
	ErrSessionExpired  = errors.New("session expired")
	ErrSessionNotFound = errors.New("session not found")
	ErrUserNotFound    = errors.New("user not found")
)

type ValidationResult struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
	Role        string
}

type AuthService struct {
	querier db.Querier
}

func NewAuthService(querier db.Querier) *AuthService {
	return &AuthService{
		querier: querier,
	}
}

func (s *AuthService) ValidateSession(ctx context.Context, token string) (*ValidationResult, error) {
	session, err := s.querier.GetSessionByToken(ctx, token)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, ErrSessionExpired
	}

	user, err := s.querier.GetUserByID(ctx, session.UserId)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return &ValidationResult{
		UserID:      user.ID,
		WorkspaceID: user.WorkspaceID,
		Role:        user.Role,
	}, nil
}
