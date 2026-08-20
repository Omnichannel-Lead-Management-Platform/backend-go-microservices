package auth

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/aarondl/authboss/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/omnichannel/auth_service/internal/db"
)

// mockQuerier implements db.Querier manually by embedding it and overriding what we need.
type mockQuerier struct {
	db.Querier
	mockGetUserByEmail func(ctx context.Context, email string) (db.User, error)
	mockCreateWorkspace func(ctx context.Context, name string) (db.Workspace, error)
	mockCreateUser func(ctx context.Context, arg db.CreateUserParams) (db.User, error)
}

func (m *mockQuerier) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	if m.mockGetUserByEmail != nil {
		return m.mockGetUserByEmail(ctx, email)
	}
	return db.User{}, sql.ErrNoRows
}

func (m *mockQuerier) CreateWorkspace(ctx context.Context, name string) (db.Workspace, error) {
	if m.mockCreateWorkspace != nil {
		return m.mockCreateWorkspace(ctx, name)
	}
	return db.Workspace{}, nil
}

func (m *mockQuerier) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
	if m.mockCreateUser != nil {
		return m.mockCreateUser(ctx, arg)
	}
	return db.User{}, nil
}

func TestServerStorer_Load(t *testing.T) {
	mockDB := &mockQuerier{
		mockGetUserByEmail: func(ctx context.Context, email string) (db.User, error) {
			if email == "test@example.com" {
				return db.User{
					ID:       uuid.MustParse("00000000-0000-0000-0000-000000000123"),
					Email:    "test@example.com",
					Password: sql.NullString{String: "hashed-pw", Valid: true},
				}, nil
			}
			return db.User{}, sql.ErrNoRows
		},
	}

	storer := NewServerStorer(mockDB)

	user, err := storer.Load(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	authbossUser := user.(*User)
	if authbossUser.Email != "test@example.com" {
		t.Errorf("expected test@example.com, got %s", authbossUser.Email)
	}

	_, err = storer.Load(context.Background(), "unknown@example.com")
	if err != authboss.ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestServerStorer_Create_AdminNoInvite(t *testing.T) {
	var workspaceCreated bool
	var userCreated db.CreateUserParams

	wsid := uuid.MustParse("00000000-0000-0000-0000-000000000789")
	uid := uuid.MustParse("00000000-0000-0000-0000-000000000111")
	
	mockDB := &mockQuerier{
		mockCreateWorkspace: func(ctx context.Context, name string) (db.Workspace, error) {
			workspaceCreated = true
			return db.Workspace{ID: wsid}, nil
		},
		mockCreateUser: func(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
			userCreated = arg
			return db.User{ID: uid}, nil
		},
	}

	storer := NewServerStorer(mockDB)

	user := &User{
		User: db.User{
			Email: "admin@example.com",
			Name: "Admin User",
			Password: sql.NullString{String: "pw", Valid: true},
		},
		Arbitrary: map[string]string{"company_name": "Test Company"},
	}

	err := storer.Create(context.Background(), user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !workspaceCreated {
		t.Error("expected a new workspace to be created")
	}

	if userCreated.Role != "admin" {
		t.Errorf("expected role admin, got %s", userCreated.Role)
	}
	if userCreated.WorkspaceID != wsid {
		t.Errorf("expected %v, got %v", wsid, userCreated.WorkspaceID)
	}
}

func TestServerStorer_Create_AgentWithInvite(t *testing.T) {
	var userCreated db.CreateUserParams

	uid := uuid.MustParse("00000000-0000-0000-0000-000000000222")
	mockDB := &mockQuerier{
		mockCreateUser: func(ctx context.Context, arg db.CreateUserParams) (db.User, error) {
			userCreated = arg
			return db.User{ID: uid}, nil
		},
		mockCreateWorkspace: func(ctx context.Context, name string) (db.Workspace, error) {
			t.Fatal("should not create workspace when registering via invite")
			return db.Workspace{}, nil
		},
	}

	storer := NewServerStorer(mockDB)

	wsid := uuid.MustParse("00000000-0000-0000-0000-000000000333")
	// Create valid invite token for wsid
	claims := jwt.MapClaims{
		"workspace_id": wsid.String(),
		"exp":          time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(JWTSecret)

	user := &User{
		User: db.User{
			Email: "agent@example.com",
			Name: "Agent User",
			Password: sql.NullString{String: "pw", Valid: true},
		},
		Arbitrary: map[string]string{"invite_token": tokenString},
	}

	err := storer.Create(context.Background(), user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if userCreated.Role != "agent" {
		t.Errorf("expected role agent, got %s", userCreated.Role)
	}
	if userCreated.WorkspaceID != wsid {
		t.Errorf("expected %v, got %v", wsid, userCreated.WorkspaceID)
	}
}
