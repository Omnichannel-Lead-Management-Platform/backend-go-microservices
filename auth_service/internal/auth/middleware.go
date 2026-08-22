package auth

import (
	"net/http"

	"github.com/aarondl/authboss/v3"
	"github.com/omnichannel/common/api"
)

// RequirePermission enforces that the authenticated user has at least one of the allowed permissions
func RequirePermission(ab *authboss.Authboss, requiredPermissions ...string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

			// Check if the user has at least one of the required permissions
			hasPermission := false
			for _, required := range requiredPermissions {
				for _, perm := range u.Permissions {
					if perm == required {
						hasPermission = true
						break
					}
				}
				if hasPermission {
					break
				}
			}

			if !hasPermission {
				api.Error(w, http.StatusForbidden, "Forbidden: insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
