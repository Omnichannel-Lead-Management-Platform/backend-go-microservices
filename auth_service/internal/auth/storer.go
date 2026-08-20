package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aarondl/authboss/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/omnichannel/auth_service/internal/db"
)

// User represents the Authboss User
type User struct {
	db.User
	Arbitrary map[string]string
}

func (u *User) GetArbitrary() map[string]string {
	return u.Arbitrary
}

func (u *User) PutArbitrary(arbitrary map[string]string) {
	u.Arbitrary = arbitrary
}

func (u *User) GetPID() string { return u.Email }
func (u *User) PutPID(pid string) { u.Email = pid }

func (u *User) GetPassword() string {
	if u.Password.Valid {
		return u.Password.String
	}
	return ""
}

func (u *User) PutPassword(password string) {
	u.Password = sql.NullString{String: password, Valid: true}
}

func (u *User) GetEmail() string {
	return u.Email
}

func (u *User) PutEmail(email string) {
	u.Email = email
}

func (u *User) GetRecoverSelector() string {
	if u.RecoverToken.Valid {
		// In a real implementation with split tokens, this would be a selector column
		return u.RecoverToken.String
	}
	return ""
}

func (u *User) PutRecoverSelector(selector string) {
	u.RecoverToken = sql.NullString{String: selector, Valid: true}
}

func (u *User) GetRecoverVerifier() string {
	return u.GetRecoverSelector() // For simplicity if using a single token
}

func (u *User) PutRecoverVerifier(verifier string) {
	// Not storing verifier separately in this simple implementation
}

func (u *User) GetRecoverExpiry() time.Time {
	if u.RecoverTokenExpiry.Valid {
		return u.RecoverTokenExpiry.Time
	}
	return time.Time{}
}

func (u *User) PutRecoverExpiry(expiry time.Time) {
	u.RecoverTokenExpiry = sql.NullTime{Time: expiry, Valid: true}
}

// ServerStorer implements authboss.ServerStorer
type ServerStorer struct {
	db db.Querier  
}

func NewServerStorer(querier db.Querier) *ServerStorer {
	return &ServerStorer{db: querier}
}

func (s *ServerStorer) Load(ctx context.Context, key string) (authboss.User, error) {
	user, err := s.db.GetUserByEmail(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, authboss.ErrUserNotFound
		}
		return nil, err
	}
	return &User{User: user}, nil
}

func (s *ServerStorer) Save(ctx context.Context, user authboss.User) error {
	u, ok := user.(*User)
	if !ok {
		return errors.New("invalid user type")
	}

	// For simplicity, we just update tokens and password. 
	// Authboss usually calls Save after Recover/Confirm actions.
	_, err := s.db.UpdateUserTokens(ctx, db.UpdateUserTokensParams{
		ID:                 u.ID,
		RecoverToken:       u.RecoverToken,
		RecoverTokenExpiry: u.RecoverTokenExpiry,
		ConfirmToken:       u.ConfirmToken,
	})
	if err != nil {
		return err
	}

	_, err = s.db.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:       u.ID,
		Password: u.Password,
	})
	
	return err
}

func (s *ServerStorer) New(ctx context.Context) authboss.User {
	return &User{
		User: db.User{
			ID: uuid.New(),
		},
	}
}

func (s *ServerStorer) Create(ctx context.Context, user authboss.User) error {
	u, ok := user.(*User)
	if !ok {
		return errors.New("invalid user type")
	}

	companyName := "Default Company"
	userName := "New User"
	
	if u.Arbitrary != nil {
		if c, ok := u.Arbitrary["company_name"]; ok && c != "" {
			companyName = c
		}
		if n, ok := u.Arbitrary["name"]; ok && n != "" {
			userName = n
		}
	}

	var workspaceID uuid.UUID
	var role = "admin" // Default to admin for new workspaces

	if u.Arbitrary != nil && u.Arbitrary["invite_token"] != "" {
		tokenStr := u.Arbitrary["invite_token"]
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return JWTSecret, nil
		})

		if err != nil || !token.Valid {
			return errors.New("invalid or expired invite token")
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if wsIDStr, ok := claims["workspace_id"].(string); ok {
				workspaceID, err = uuid.Parse(wsIDStr)
				if err == nil {
					role = "agent" // Invited users are agents
				}
			}
		}

		if workspaceID == uuid.Nil {
			return errors.New("invalid workspace ID in invite token")
		}
	} else {
		// Create workspace first since no invite token was provided
		createdWorkspace, err := s.db.CreateWorkspace(ctx, companyName)
		if err != nil {
			return err
		}
		workspaceID = createdWorkspace.ID
	}

	arg := db.CreateUserParams{
		WorkspaceID:   workspaceID, 
		Role:          role,
		Name:          userName,
		Email:         u.Email,
		Password:      u.Password,
		EmailVerified: false,
	}

	createdUser, err := s.db.CreateUser(ctx, arg)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value") || strings.Contains(err.Error(), "unique constraint") {
			return authboss.ErrUserFound
		}
		return err
	}
	u.ID = createdUser.ID
	return nil
}
