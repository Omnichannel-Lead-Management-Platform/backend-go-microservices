package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aarondl/authboss/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/omnichannel/common/logger"
	"go.uber.org/zap"
)

var JWTSecret = []byte("super-secret-jwt-key-replace-in-prod")

// JWTClientState represents the parsed JWT data
type JWTClientState struct {
	data map[string]string
}

func (j *JWTClientState) Get(key string) (string, bool) {
	val, ok := j.data[key]
	return val, ok
}

// TokenBlacklister defines the interface for blacklisting JWT tokens
type TokenBlacklister interface {
	Blacklist(ctx context.Context, token string, expiration time.Duration) error
	IsBlacklisted(ctx context.Context, token string) bool
}

// JWTReadWriter implements authboss.ClientStateReadWriter
type JWTReadWriter struct {
	blacklister TokenBlacklister
}

func NewJWTReadWriter(b TokenBlacklister) *JWTReadWriter {
	return &JWTReadWriter{
		blacklister: b,
	}
}

// ReadState reads the JWT from the Authorization header
func (j *JWTReadWriter) ReadState(r *http.Request) (authboss.ClientState, error) {
	state := &JWTClientState{data: make(map[string]string)}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return state, nil // Empty state if no token
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return JWTSecret, nil
	})

	if err != nil || !token.Valid {
		return state, nil
	}

	// Check if token is blacklisted
	if j.blacklister != nil && j.blacklister.IsBlacklisted(r.Context(), tokenString) {
		return state, nil // Treat as logged out
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if sess, ok := claims["sess"].(map[string]interface{}); ok {
			for k, v := range sess {
				if str, ok := v.(string); ok {
					state.data[k] = str
				}
			}
		}
	}

	return state, nil
}

// WriteState creates a new JWT and adds it to the HTTP response header
func (j *JWTReadWriter) WriteState(w http.ResponseWriter, state authboss.ClientState, ev []authboss.ClientStateEvent) error {
	logger.Log.Debug("WriteState called", zap.Int("events", len(ev)))
	jwtState, ok := state.(*JWTClientState)
	if !ok {
		// If it's a completely new state, initialize it
		jwtState = &JWTClientState{data: make(map[string]string)}
	}

	for _, e := range ev {
		if e.Kind == authboss.ClientStateEventPut {
			jwtState.data[e.Key] = e.Value
		} else if e.Kind == authboss.ClientStateEventDel {
			delete(jwtState.data, e.Key)
		} else if e.Kind == authboss.ClientStateEventDelAll {
			jwtState.data = make(map[string]string)
		}
	}

	// Generate new JWT
	claims := jwt.MapClaims{
		"sess": jwtState.data,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
	}
	
	// If permissions are in session, hoist them to a top-level array for easier parsing by other services
	if permsStr, ok := jwtState.data["permissions"]; ok {
		claims["permissions"] = strings.Split(permsStr, ",")
		// Optionally remove from sess to save space
		// delete(jwtState.data, "permissions")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(JWTSecret)
	if err != nil {
		return err
	}

	// We expose the token to the frontend via a custom header
	w.Header().Set("X-Access-Token", tokenString)
	// And we allow JS to read it
	w.Header().Add("Access-Control-Expose-Headers", "X-Access-Token")

	return nil
}
