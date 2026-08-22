package auth

import (
	"net/http"
	"time"

	"github.com/aarondl/authboss/v3"
	"github.com/golang-jwt/jwt/v5"
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
