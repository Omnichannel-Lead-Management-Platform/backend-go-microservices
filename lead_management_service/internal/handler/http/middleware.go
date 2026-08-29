package http

import (
	"context"
	"net/http"
)

// Define custom types for context keys to avoid collisions
type contextKey string

const (
	WorkspaceIDKey contextKey = "workspace_id"
	UserIDKey      contextKey = "user_id"
)

// MockAuthMiddleware is a temporary security guard.
func MockAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// 1. Look at the incoming request's headers
		workspaceID := r.Header.Get("X-Workspace-ID")
		userID := r.Header.Get("X-User-ID")

		// 2. If the user didn't provide a Workspace ID, block them!
		if workspaceID == "" {
			http.Error(w, "Unauthorized: X-Workspace-ID header is missing", http.StatusUnauthorized)
			return
		}

		// 3. Put the ID securely into the Request Context so our Handlers can use it
		ctx := context.WithValue(r.Context(), WorkspaceIDKey, workspaceID)
		ctx = context.WithValue(ctx, UserIDKey, userID)

		// 4. Let the user pass through to the actual URL handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
