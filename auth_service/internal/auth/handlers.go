package auth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/aarondl/authboss/v3"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/omnichannel/auth_service/internal/db"
	"github.com/omnichannel/common/api"
)

func MeHandler(ab *authboss.Authboss) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := ab.CurrentUser(r)
		if err != nil || user == nil {
			api.Error(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		u, ok := user.(*User)
		if !ok {
			api.Error(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}

		// u.User is our internal/db.User struct which contains Name, RoleID, WorkspaceID, Email, etc.
		// We omit the password hash.
		profile := map[string]interface{}{
			"id":           u.ID,
			"email":        u.Email,
			"name":         u.Name,
			"role_id":      u.RoleID.UUID,
			"permissions":  u.Permissions,
			"workspace_id": u.WorkspaceID,
		}

		api.Success(w, profile, "User profile retrieved successfully")
	}
}

func GenerateInviteHandler(ab *authboss.Authboss) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := ab.CurrentUser(r)
		if err != nil || user == nil {
			api.Error(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		u, ok := user.(*User)
		if !ok {
			api.Error(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}

		// Create an invite token containing the workspace ID
		claims := jwt.MapClaims{
			"workspace_id": u.WorkspaceID,
			"exp":          time.Now().Add(7 * 24 * time.Hour).Unix(), // 7 days expiration
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(JWTSecret)
		if err != nil {
			api.Error(w, http.StatusInternalServerError, "Failed to generate invite token")
			return
		}

		responseData := map[string]interface{}{
			"invite_token": tokenString,
			"invite_link":  "http://localhost:5173/register?invite_token=" + tokenString, // Adjust domain for production
		}

		api.Success(w, responseData, "Invite link generated successfully")
	}
}

func CreateRoleHandler(ab *authboss.Authboss, querier db.Querier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := ab.CurrentUser(r)
		if err != nil || user == nil {
			api.Error(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		u, _ := user.(*User)

		var req struct {
			Name        string   "json:\"name\""
			Permissions []string "json:\"permissions\""
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			api.Error(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		ctx := r.Context()
		role, err := querier.CreateRole(ctx, db.CreateRoleParams{
			WorkspaceID: u.WorkspaceID,
			Name:        req.Name,
		})
		if err != nil {
			api.Error(w, http.StatusInternalServerError, "Failed to create role")
			return
		}

		for _, permName := range req.Permissions {
			perm, err := querier.GetPermissionByName(ctx, permName)
			if err == nil {
				querier.AssignPermissionToRole(ctx, db.AssignPermissionToRoleParams{
					RoleID:       role.ID,
					PermissionID: perm.ID,
				})
			}
		}

		api.Success(w, role, "Role created successfully")
	}
}

func ListUsersHandler(ab *authboss.Authboss, querier db.Querier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := ab.CurrentUser(r)
		if err != nil || user == nil {
			api.Error(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		u, _ := user.(*User)

		users, err := querier.GetUsersByWorkspaceID(r.Context(), u.WorkspaceID)
		if err != nil {
			api.Error(w, http.StatusInternalServerError, "Failed to retrieve users")
			return
		}

		api.Success(w, users, "Users retrieved successfully")
	}
}

func UpdateUserRoleHandler(ab *authboss.Authboss, querier db.Querier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := ab.CurrentUser(r)
		if err != nil || user == nil {
			api.Error(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		u, _ := user.(*User)

		userIDStr := chi.URLParam(r, "id")
		targetUserID, err := uuid.Parse(userIDStr)
		if err != nil {
			api.Error(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		var req struct {
			RoleID string "json:\"role_id\""
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			api.Error(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		targetRoleID, err := uuid.Parse(req.RoleID)
		if err != nil {
			api.Error(w, http.StatusBadRequest, "Invalid role ID")
			return
		}

		err = querier.UpdateUserRole(r.Context(), db.UpdateUserRoleParams{
			ID:          targetUserID,
			RoleID:      uuid.NullUUID{UUID: targetRoleID, Valid: true},
			WorkspaceID: u.WorkspaceID,
		})
		if err != nil {
			api.Error(w, http.StatusInternalServerError, "Failed to update user role")
			return
		}

		api.Success(w, nil, "User role updated successfully")
	}
}

