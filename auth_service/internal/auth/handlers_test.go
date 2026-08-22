package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aarondl/authboss/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/omnichannel/auth_service/internal/db"
)

func TestMeHandler(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/me", nil)

	// Mock authenticated user
	uid := uuid.MustParse("00000000-0000-0000-0000-000000000123")
	wsid := uuid.MustParse("00000000-0000-0000-0000-000000000456")
	user := &User{
		User: db.User{
			ID:          uid,
			Email:       "test@example.com",
			Name:        "Test User",
			RoleID:      uuid.NullUUID{UUID: uuid.MustParse("22222222-2222-2222-2222-222222222222"), Valid: true},
			WorkspaceID: wsid,
		},
		Permissions: []string{"leads:read"},
	}

	// Inject user into context
	ctx := context.WithValue(r.Context(), authboss.CTXKeyUser, user)
	r = r.WithContext(ctx)

	// Call handler
	ab := authboss.New()
	handler := MeHandler(ab)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("expected data object in response")
	}

	if data["email"] != "test@example.com" {
		t.Errorf("expected test@example.com, got %v", data["email"])
	}
	if data["role_id"] != "22222222-2222-2222-2222-222222222222" {
		t.Errorf("expected role_id 22222222-2222-2222-2222-222222222222, got %v", data["role_id"])
	}
	
	// Test permissions array in response
	perms, ok := data["permissions"].([]interface{})
	if !ok || len(perms) == 0 || perms[0] != "leads:read" {
		t.Errorf("expected permissions [leads:read], got %v", data["permissions"])
	}
	if data["workspace_id"] != wsid.String() {
		t.Errorf("expected %s, got %v", wsid.String(), data["workspace_id"])
	}
}

func TestGenerateInviteHandler(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/invite", nil)

	// Mock authenticated user (must be admin)
	wsid := uuid.MustParse("00000000-0000-0000-0000-000000000789")
	user := &User{
		User: db.User{
			WorkspaceID: wsid,
		},
	}

	ctx := context.WithValue(r.Context(), authboss.CTXKeyUser, user)
	r = r.WithContext(ctx)

	ab := authboss.New()
	handler := GenerateInviteHandler(ab)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("expected data object in response")
	}

	inviteToken, ok := data["invite_token"].(string)
	if !ok || inviteToken == "" {
		t.Fatal("expected invite_token string in response")
	}

	// Verify the token
	parsedToken, err := jwt.Parse(inviteToken, func(token *jwt.Token) (interface{}, error) {
		return JWTSecret, nil
	})
	
	if err != nil || !parsedToken.Valid {
		t.Fatalf("invalid invite token generated: %v", err)
	}

	claims := parsedToken.Claims.(jwt.MapClaims)
	if claims["workspace_id"] != wsid.String() {
		t.Errorf("expected workspace_id %s in token, got %v", wsid.String(), claims["workspace_id"])
	}
	
	// Check expiration is roughly 7 days
	exp, ok := claims["exp"].(float64)
	if !ok {
		t.Fatal("expected exp claim")
	}
	
	expectedExp := time.Now().Add(7 * 24 * time.Hour).Unix()
	if int64(exp) < expectedExp-100 || int64(exp) > expectedExp+100 {
		t.Errorf("expected exp around %d, got %v", expectedExp, exp)
	}
}
