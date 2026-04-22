package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/asdlc/leave-api/internal/auth"
)

type contextKey string

const (
	CtxUserID contextKey = "userId"
	CtxRole   contextKey = "role"
	CtxName   contextKey = "name"
	CtxToken  contextKey = "renewedToken"
)

type Principal struct {
	UserID string
	Role   string
	Name   string
}

func GetPrincipal(r *http.Request) (Principal, bool) {
	uid, _ := r.Context().Value(CtxUserID).(string)
	role, _ := r.Context().Value(CtxRole).(string)
	name, _ := r.Context().Value(CtxName).(string)
	if uid == "" {
		return Principal{}, false
	}
	return Principal{UserID: uid, Role: role, Name: name}, true
}

// CORS adds permissive CORS headers. Preflight short-circuits with 204.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Service-Token")
		w.Header().Set("Access-Control-Expose-Headers", "X-Renewed-Token")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Auth enforces JWT on protected endpoints. Issues a fresh sliding token per request.
func Auth(signer *auth.Signer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeJSONErr(w, http.StatusUnauthorized, "missing bearer token")
				return
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := signer.Verify(token)
			if err != nil {
				writeJSONErr(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}
			// Slide expiry: issue a new token and expose via header.
			renewed := signer.Issue(claims.Sub, claims.Role, claims.Name)
			w.Header().Set("X-Renewed-Token", renewed)
			ctx := context.WithValue(r.Context(), CtxUserID, claims.Sub)
			ctx = context.WithValue(ctx, CtxRole, claims.Role)
			ctx = context.WithValue(ctx, CtxName, claims.Name)
			ctx = context.WithValue(ctx, CtxToken, renewed)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole enforces one of the allowed roles.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(CtxRole).(string)
			for _, allowed := range roles {
				if role == allowed {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeJSONErr(w, http.StatusForbidden, "role not permitted")
		})
	}
}

// ServiceToken allows the notification-dispatcher to bypass user JWT with a shared token.
// Accepts either X-Service-Token header or a bearer JWT of a valid user.
func ServiceTokenOrAuth(signer *auth.Signer, serviceToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			svc := r.Header.Get("X-Service-Token")
			if svc != "" && svc == serviceToken {
				ctx := context.WithValue(r.Context(), CtxUserID, "service")
				ctx = context.WithValue(ctx, CtxRole, "service")
				ctx = context.WithValue(ctx, CtxName, "notification-dispatcher")
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			Auth(signer)(next).ServeHTTP(w, r)
		})
	}
}

func writeJSONErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
