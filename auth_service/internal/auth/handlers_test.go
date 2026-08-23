package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aarondl/authboss/v3"
	"github.com/go-chi/chi/v5"
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

func TestCreateRoleHandler(t *testing.T) {
	ab := authboss.New()
	wsid := uuid.MustParse("00000000-0000-0000-0000-000000000555")
	
	mockDB := &mockQuerier{
		mockCreateRole: func(ctx context.Context, arg db.CreateRoleParams) (db.Role, error) {
			if arg.WorkspaceID != wsid {
				t.Errorf("expected workspace id %v, got %v", wsid, arg.WorkspaceID)
			}
			return db.Role{ID: uuid.New(), Name: arg.Name, WorkspaceID: arg.WorkspaceID}, nil
		},
		mockGetPermissionByName: func(ctx context.Context, name string) (db.Permission, error) {
			return db.Permission{ID: uuid.New(), Name: name}, nil
		},
		mockAssignPermissionToRole: func(ctx context.Context, arg db.AssignPermissionToRoleParams) error {
			return nil
		},
	}

	handler := CreateRoleHandler(ab, mockDB)

	user := &User{
		User: db.User{
			ID:          uuid.New(),
			WorkspaceID: wsid,
		},
	}

	body := bytes.NewBufferString("{\"name\": \"Manager\", \"permissions\": [\"leads:read\"]}")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/auth/roles", body)
	
	ctx := context.WithValue(r.Context(), authboss.CTXKeyUser, user)
	r = r.WithContext(ctx)

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}
}

func TestListUsersHandler(t *testing.T) {
	ab := authboss.New()
	wsid := uuid.MustParse("00000000-0000-0000-0000-000000000555")
	
	mockDB := &mockQuerier{
		mockGetUsersByWorkspaceID: func(ctx context.Context, workspaceID uuid.UUID) ([]db.GetUsersByWorkspaceIDRow, error) {
			if workspaceID != wsid {
				t.Errorf("expected workspace id %v, got %v", wsid, workspaceID)
			}
			return []db.GetUsersByWorkspaceIDRow{
				{ID: uuid.New(), Name: "Alice"},
				{ID: uuid.New(), Name: "Bob"},
			}, nil
		},
	}

	handler := ListUsersHandler(ab, mockDB)

	user := &User{
		User: db.User{
			ID:          uuid.New(),
			WorkspaceID: wsid,
		},
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/auth/users", nil)
	
	ctx := context.WithValue(r.Context(), authboss.CTXKeyUser, user)
	r = r.WithContext(ctx)

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}
}

func TestUpdateUserRoleHandler(t *testing.T) {
	ab := authboss.New()
	wsid := uuid.MustParse("00000000-0000-0000-0000-000000000555")
	targetUserID := uuid.New()
	targetRoleID := uuid.New()
	
	mockDB := &mockQuerier{
		mockUpdateUserRole: func(ctx context.Context, arg db.UpdateUserRoleParams) error {
			if arg.WorkspaceID != wsid {
				t.Errorf("expected workspace id %v, got %v", wsid, arg.WorkspaceID)
			}
			if arg.ID != targetUserID {
				t.Errorf("expected user id %v, got %v", targetUserID, arg.ID)
			}
			if arg.RoleID.UUID != targetRoleID {
				t.Errorf("expected role id %v, got %v", targetRoleID, arg.RoleID.UUID)
			}
			return nil
		},
	}

	handler := UpdateUserRoleHandler(ab, mockDB)

	user := &User{
		User: db.User{
			ID:          uuid.New(),
			WorkspaceID: wsid,
		},
	}

	body := bytes.NewBufferString("{\"role_id\": \"" + targetRoleID.String() + "\"}")
	w := httptest.NewRecorder()
	
	// Since chi is used, we need to mock chi routing context
	r := httptest.NewRequest("PUT", "/api/auth/users/"+targetUserID.String()+"/role", body)
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", targetUserID.String())
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx)
	
	ctx = context.WithValue(ctx, authboss.CTXKeyUser, user)
	r = r.WithContext(ctx)

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}
}


func TestIntrospectHandler(t *testing.T) {
	ab := authboss.New()
	ab.Config.Storage.SessionState = NewJWTReadWriter(nil)

	// Valid token test
	claims := jwt.MapClaims{
		"sess": map[string]interface{}{
			"uid":          "testuser@example.com",
			"workspace_id": "ws1",
			"role_id":      "r1",
			"permissions":  "users:manage,roles:manage",
		},
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(JWTSecret)

	req := httptest.NewRequest("GET", "/api/auth/introspect", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	handler := IntrospectHandler(ab)
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}

	var resp IntrospectResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Active {
		t.Error("expected Active to be true")
	}
	if resp.UserID != "testuser@example.com" {
		t.Errorf("expected UserID testuser@example.com, got %s", resp.UserID)
	}
	if len(resp.Permissions) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(resp.Permissions))
	}

	// Invalid token test
	reqInvalid := httptest.NewRequest("GET", "/api/auth/introspect", nil)
	reqInvalid.Header.Set("Authorization", "Bearer badtoken")
	wInvalid := httptest.NewRecorder()

	handler.ServeHTTP(wInvalid, reqInvalid)

	if wInvalid.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for invalid token, got %d", wInvalid.Code)
	}
}
