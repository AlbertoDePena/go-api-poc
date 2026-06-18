package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

type ctxKey string

const claimsKey ctxKey = "claims"

// Claims we care about from an Entra ID access token.
type tokenClaims struct {
	Scope string   `json:"scp"`   // delegated permissions (space-separated)
	Roles []string `json:"roles"` // application permissions
	Name  string   `json:"name"`
	OID   string   `json:"oid"`
}

func RequireAuth(verifier *oidc.IDTokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authz := r.Header.Get("Authorization")
			if !strings.HasPrefix(authz, "Bearer ") {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}
			raw := strings.TrimPrefix(authz, "Bearer ")

			// Verifies signature (via JWKS), issuer, audience, and expiry.
			tok, err := verifier.Verify(r.Context(), raw)
			if err != nil {
				slog.ErrorContext(r.Context(), "token verification failed", "error", err)
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			var claims tokenClaims
			if err := tok.Claims(&claims); err != nil {
				slog.ErrorContext(r.Context(), "token claims parsing failed", "error", err)
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, &claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := r.Context().Value(claimsKey).(*tokenClaims)
			if slices.Contains(strings.Fields(claims.Scope), scope) {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "insufficient scope", http.StatusForbidden)
		})
	}
}

func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := r.Context().Value(claimsKey).(*tokenClaims)
			if slices.Contains(claims.Roles, role) {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "missing required role", http.StatusForbidden)
		})
	}
}
