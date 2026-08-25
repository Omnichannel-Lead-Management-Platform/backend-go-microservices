package auth

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/aarondl/authboss/v3"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/omnichannel/auth_service/internal/db"
	"github.com/sqlc-dev/pqtype"
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
		var responseUsers []map[string]interface{}
		for _, dbUser := range users {
			uMap := map[string]interface{}{
				"id":            dbUser.ID,
				"workspace_id":  dbUser.WorkspaceID,
				"name":          dbUser.Name,
				"email":         dbUser.Email,
				"emailVerified": dbUser.EmailVerified,
			}
			
			if dbUser.RoleID.Valid {
				uMap["role_id"] = dbUser.RoleID.UUID.String()
			} else {
				uMap["role_id"] = nil
			}

			if dbUser.Image.Valid {
				uMap["image"] = dbUser.Image.String
			} else {
				uMap["image"] = nil
			}

			if dbUser.CreatedAt.Valid {
				uMap["createdAt"] = dbUser.CreatedAt.Time
			}

			responseUsers = append(responseUsers, uMap)
		}

		api.Success(w, responseUsers, "Users retrieved successfully")
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

func GetRolesHandler(ab *authboss.Authboss, querier db.Querier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := ab.CurrentUser(r)
		if err != nil || user == nil {
			api.Error(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		u, _ := user.(*User)

		roles, err := querier.GetRolesByWorkspaceID(r.Context(), u.WorkspaceID)
		if err != nil {
			api.Error(w, http.StatusInternalServerError, "Failed to retrieve roles")
			return
		}

		api.Success(w, roles, "Roles retrieved successfully")
	}
}

func GetPermissionsHandler(ab *authboss.Authboss, querier db.Querier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		perms, err := querier.GetAllPermissions(r.Context())
		if err != nil {
			api.Error(w, http.StatusInternalServerError, "Failed to retrieve permissions")
			return
		}

		api.Success(w, perms, "Permissions retrieved successfully")
	}
}

func GetRolePermissionsHandler(ab *authboss.Authboss, querier db.Querier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		roleIDStr := chi.URLParam(r, "id")
		roleID, err := uuid.Parse(roleIDStr)
		if err != nil {
			api.Error(w, http.StatusBadRequest, "Invalid role ID")
			return
		}

		perms, err := querier.GetRolePermissions(r.Context(), roleID)
		if err != nil {
			api.Error(w, http.StatusInternalServerError, "Failed to retrieve role permissions")
			return
		}

		api.Success(w, perms, "Role permissions retrieved successfully")
	}
}



// IntrospectResponse is the JSON payload returned by the introspection endpoint
type IntrospectResponse struct {
	Active      bool     `json:"active"`
	UserID      string   `json:"user_id,omitempty"`
	WorkspaceID string   `json:"workspace_id,omitempty"`
	RoleID      string   `json:"role_id,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

// IntrospectHandler validates a JWT without hitting the database (purely based on JWT claims).
// This is used by the API Gateway to authorize requests to other microservices.
func IntrospectHandler(ab *authboss.Authboss) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Attempt to read the session state (this parses the JWT and checks the Redis blacklist)
		state, err := ab.Storage.SessionState.ReadState(r)
		if err != nil || state == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(IntrospectResponse{Active: false})
			return
		}

		uid, hasUid := state.Get(authboss.SessionKey)
		workspaceID, hasWorkspace := state.Get("workspace_id")
		
		if !hasUid || !hasWorkspace {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(IntrospectResponse{Active: false})
			return
		}

		roleID, _ := state.Get("role_id")
		permsStr, _ := state.Get("permissions")
		
		var perms []string
		if permsStr != "" {
			perms = strings.Split(permsStr, ",")
		}

		resp := IntrospectResponse{
			Active:      true,
			UserID:      uid,
			WorkspaceID: workspaceID,
			RoleID:      roleID,
			Permissions: perms,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// UpdateWorkspaceRequest represents the payload for updating workspace settings
type UpdateWorkspaceRequest struct {
	Name     string          `json:"name"`
	Settings json.RawMessage `json:"settings"`
}

// GetWorkspaceHandler returns the current workspace details
func GetWorkspaceHandler(ab *authboss.Authboss, querier db.Querier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := ab.CurrentUser(r)
		if err != nil || u == nil {
			api.Error(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		user, ok := u.(*User)
		if !ok {
			api.Error(w, http.StatusInternalServerError, "Invalid user type")
			return
		}

		workspace, err := querier.GetWorkspaceByID(r.Context(), user.WorkspaceID)
		if err != nil {
			api.Error(w, http.StatusInternalServerError, "Failed to retrieve workspace")
			return
		}

		api.Success(w, workspace, "Workspace retrieved successfully")
	}
}

// UpdateWorkspaceHandler allows an admin to update workspace details
func UpdateWorkspaceHandler(ab *authboss.Authboss, querier db.Querier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := ab.CurrentUser(r)
		if err != nil || u == nil {
			api.Error(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		user, ok := u.(*User)
		if !ok {
			api.Error(w, http.StatusInternalServerError, "Invalid user type")
			return
		}

		var req UpdateWorkspaceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			api.Error(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		if req.Name == "" {
			api.Error(w, http.StatusBadRequest, "Workspace name is required")
			return
		}
		
		if len(req.Settings) == 0 {
			req.Settings = json.RawMessage(`{}`)
		}

		workspace, err := querier.UpdateWorkspace(r.Context(), db.UpdateWorkspaceParams{
			ID:       user.WorkspaceID,
			Name:     req.Name,
			Settings: pqtype.NullRawMessage{RawMessage: req.Settings, Valid: true},
		})
		
		if err != nil {
			api.Error(w, http.StatusInternalServerError, "Failed to update workspace")
			return
		}

		api.Success(w, workspace, "Workspace updated successfully")
	}
}
