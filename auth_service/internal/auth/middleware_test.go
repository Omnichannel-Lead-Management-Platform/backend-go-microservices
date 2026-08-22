package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aarondl/authboss/v3"
	"github.com/omnichannel/auth_service/internal/db"
)

func TestRequirePermission_AdminAccess(t *testing.T) {
	ab := authboss.New()
	middleware := RequirePermission(ab, "users:manage")
	
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/protected", nil)

	user := &User{
		User:        db.User{},
		Permissions: []string{"users:manage", "leads:read"},
	}
	ctx := context.WithValue(r.Context(), authboss.CTXKeyUser, user)
	r = r.WithContext(ctx)

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for user with permission, got %d", w.Code)
	}
	if w.Body.String() != "success" {
		t.Errorf("expected success body, got %s", w.Body.String())
	}
}

func TestRequirePermission_AgentDenied(t *testing.T) {
	ab := authboss.New()
	middleware := RequirePermission(ab, "users:manage")
	
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/protected", nil)

	user := &User{
		User:        db.User{},
		Permissions: []string{"leads:read"},
	}
	ctx := context.WithValue(r.Context(), authboss.CTXKeyUser, user)
	r = r.WithContext(ctx)

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for user without permission, got %d", w.Code)
	}
}

func TestRequirePermission_NoUser(t *testing.T) {
	ab := authboss.New()
	middleware := RequirePermission(ab, "users:manage")
	
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/protected", nil)
	// No user in context

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for no user, got %d", w.Code)
	}
}
