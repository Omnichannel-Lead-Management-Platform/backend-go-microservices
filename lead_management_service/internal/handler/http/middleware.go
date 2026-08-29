package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Define custom types for context keys to avoid collisions
type contextKey string

const (
	WorkspaceIDKey contextKey = "workspace_id"
	UserIDKey      contextKey = "user_id"
	PermissionsKey contextKey = "permissions"
)

// IntrospectResponse matches the JSON returned by the Auth Service
type IntrospectResponse struct {
	Active      bool     `json:"active"`
	UserID      string   `json:"user_id"`
	WorkspaceID string   `json:"workspace_id"`
	RoleID      string   `json:"role_id"`
	Permissions []string `json:"permissions"`
}

// AuthMiddleware intercepts the request, calls the Auth Service, and injects data into the Context.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Get the Bearer token from the incoming request
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized: Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// 2. Call the Auth Service Introspect endpoint
		// In production, this URL would come from an environment variable (e.g., os.Getenv("AUTH_SERVICE_URL"))
		authURL := "http://localhost:8080/api/auth/introspect"
		
		req, err := http.NewRequestWithContext(r.Context(), "GET", authURL, nil)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		
		// Pass the token forward to the Auth Service
		req.Header.Set("Authorization", authHeader)
		
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			http.Error(w, "Unauthorized: Invalid or expired token", http.StatusUnauthorized)
			return
		}
		defer resp.Body.Close()

		// 3. Decode the Auth Service response
		var introspect IntrospectResponse
		if err := json.NewDecoder(resp.Body).Decode(&introspect); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if !introspect.Active {
			http.Error(w, "Unauthorized: Token is inactive", http.StatusUnauthorized)
			return
		}

		// 4. Put the verified data securely into the Request Context
		ctx := context.WithValue(r.Context(), WorkspaceIDKey, introspect.WorkspaceID)
		ctx = context.WithValue(ctx, UserIDKey, introspect.UserID)
		ctx = context.WithValue(ctx, PermissionsKey, introspect.Permissions)

		// 5. Let the user pass through to the actual URL handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission is a powerful middleware that enforces Role-Based Access Control!
func RequirePermission(requiredPermission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract permissions array from the context (injected by AuthMiddleware)
			permissions, ok := r.Context().Value(PermissionsKey).([]string)
			if !ok {
				http.Error(w, "Forbidden: No permissions found", http.StatusForbidden)
				return
			}

			// Check if the user has the required permission
			hasPermission := false
			for _, p := range permissions {
				if p == requiredPermission {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				msg := fmt.Sprintf("Forbidden: You lack the required permission '%s'", requiredPermission)
				http.Error(w, msg, http.StatusForbidden)
				return
			}

			// User is authorized! Pass them through.
			next.ServeHTTP(w, r)
		})
	}
}
