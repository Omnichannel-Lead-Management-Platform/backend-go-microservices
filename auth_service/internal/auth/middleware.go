package auth

import (
	"net/http"

	"github.com/aarondl/authboss/v3"
	"github.com/omnichannel/common/api"
)

// RequireRole enforces that the authenticated user has one of the allowed roles
func RequireRole(ab *authboss.Authboss, allowedRoles ...string) func(next http.Handler) http.Handler {
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

			// Check if the user's role is in the list of allowed roles
			hasRole := false
			for _, role := range allowedRoles {
				if u.Role == role {
					hasRole = true
					break
				}
			}

			if !hasRole {
				api.Error(w, http.StatusForbidden, "Forbidden: insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
