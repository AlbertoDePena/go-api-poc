package middleware

import (
	"net/http"

	"github.com/craneww/api-poc/internal/core/feature"
)

func RequireFeature(fm feature.FeatureManager, feature string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !fm.IsEnabled(r.Context(), feature) {
				http.NotFound(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
