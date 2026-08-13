package auth

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/aarondl/authboss/v3"

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

func (u *User) GetRecoverToken() string {
	if u.RecoverToken.Valid {
		return u.RecoverToken.String
	}
	return ""
}

func (u *User) PutRecoverToken(token string) {
	u.RecoverToken = sql.NullString{String: token, Valid: true}
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

	// Create workspace first
	createdWorkspace, err := s.db.CreateWorkspace(ctx, companyName)
	if err != nil {
		return err
	}

	arg := db.CreateUserParams{
		WorkspaceID:   createdWorkspace.ID, 
		Role:          "admin", // First user of a workspace is an admin
		Name:          userName,
		Email:         u.Email,
		Password:      u.Password,
		EmailVerified: false,
	}

	createdUser, err := s.db.CreateUser(ctx, arg)
	if err != nil {
		return err
	}
	u.ID = createdUser.ID
	return nil
}
