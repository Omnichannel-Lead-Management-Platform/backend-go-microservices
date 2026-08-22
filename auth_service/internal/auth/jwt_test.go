package auth

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aarondl/authboss/v3"
	"github.com/golang-jwt/jwt/v5"
)

func TestJWTReadWriter_WriteState(t *testing.T) {
	rw := NewJWTReadWriter(nil)
	w := httptest.NewRecorder()

	state := &JWTClientState{data: make(map[string]string)}
	events := []authboss.ClientStateEvent{
		{Kind: authboss.ClientStateEventPut, Key: "uid", Value: "testuser@example.com"},
	}

	err := rw.WriteState(w, state, events)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	token := w.Header().Get("X-Access-Token")
	if token == "" {
		t.Fatal("expected X-Access-Token header to be set")
	}

	// Verify token
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return JWTSecret, nil
	})
	if err != nil || !parsedToken.Valid {
		t.Fatalf("invalid token generated: %v", err)
	}

	claims := parsedToken.Claims.(jwt.MapClaims)
	sess := claims["sess"].(map[string]interface{})
	if sess["uid"] != "testuser@example.com" {
		t.Errorf("expected uid=testuser@example.com, got %v", sess["uid"])
	}
}

func TestJWTReadWriter_ReadState(t *testing.T) {
	rw := NewJWTReadWriter(nil)
	
	// Create a valid token
	claims := jwt.MapClaims{
		"sess": map[string]string{"uid": "testuser@example.com"},
		"exp":  time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(JWTSecret)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	state, err := rw.ReadState(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	uid, ok := state.Get("uid")
	if !ok || uid != "testuser@example.com" {
		t.Errorf("expected uid=testuser@example.com, got %v", uid)
	}
}

func TestJWTReadWriter_ReadState_Invalid(t *testing.T) {
	rw := NewJWTReadWriter(nil)
	
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")

	state, err := rw.ReadState(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, ok := state.Get("uid")
	if ok {
		t.Error("expected empty state for invalid token")
	}
}

type mockTokenBlacklister struct {
	blacklisted bool
}

func (m *mockTokenBlacklister) Blacklist(ctx context.Context, token string, expiration time.Duration) error {
	return nil
}

func (m *mockTokenBlacklister) IsBlacklisted(ctx context.Context, token string) bool {
	return m.blacklisted
}

func TestJWTReadWriter_ReadState_Blacklisted(t *testing.T) {
	rw := NewJWTReadWriter(&mockTokenBlacklister{blacklisted: true})
	
	claims := jwt.MapClaims{
		"sess": map[string]string{"uid": "testuser@example.com"},
		"exp":  time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(JWTSecret)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	state, err := rw.ReadState(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, ok := state.Get("uid")
	if ok {
		t.Error("expected empty state for blacklisted token")
	}
}

