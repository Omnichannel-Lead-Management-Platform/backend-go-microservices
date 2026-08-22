package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aarondl/authboss/v3"
)

func TestAPIResponder_Respond_ValidationError(t *testing.T) {
	responder := NewAPIResponder()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/register", nil)

	data := authboss.HTMLData{
		"errors": map[string][]string{
			"email": {"has already been taken"},
		},
	}

	err := responder.Respond(w, r, http.StatusOK, "register", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["message"] != "email: has already been taken" {
		t.Errorf("expected validation message, got %v", resp["message"])
	}
}

func TestAPIResponder_Respond_AuthError(t *testing.T) {
	responder := NewAPIResponder()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/login", nil)

	data := authboss.HTMLData{
		authboss.DataErr: "invalid credentials",
	}

	err := responder.Respond(w, r, http.StatusOK, "login", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["message"] != "invalid credentials" {
		t.Errorf("expected invalid credentials message, got %v", resp["message"])
	}
}

func TestAPIResponder_Respond_SuccessWithToken(t *testing.T) {
	responder := NewAPIResponder()
	w := httptest.NewRecorder()
	w.Header().Set("X-Access-Token", "fake.jwt.token")
	
	r := httptest.NewRequest("POST", "/login", nil)
	data := authboss.HTMLData{
		authboss.FlashSuccessKey: "Login successful",
	}

	err := responder.Respond(w, r, http.StatusOK, "login", data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["message"] != "Login successful" {
		t.Errorf("expected Login successful message, got %v", resp["message"])
	}

	dataMap, ok := resp["data"].(map[string]interface{})
	if !ok || dataMap["token"] != "fake.jwt.token" {
		t.Errorf("expected token in response data, got %v", dataMap["token"])
	}
}

func TestAPIResponder_Redirect(t *testing.T) {
	responder := NewAPIResponder()
	w := httptest.NewRecorder()
	w.Header().Set("X-Access-Token", "redirect.jwt.token")
	
	r := httptest.NewRequest("POST", "/register", nil)
	opts := authboss.RedirectOptions{
		Success: "Registration complete",
	}

	err := responder.Redirect(w, r, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["message"] != "Registration complete" {
		t.Errorf("expected Registration complete message, got %v", resp["message"])
	}

	dataMap, ok := resp["data"].(map[string]interface{})
	if !ok || dataMap["token"] != "redirect.jwt.token" {
		t.Errorf("expected token in response data, got %v", dataMap["token"])
	}
}
