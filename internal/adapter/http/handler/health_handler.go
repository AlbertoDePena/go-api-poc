package handler

import (
	"context"
	"net/http"
)

// Pinger checks whether a dependency is reachable.
type Pinger interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	pinger Pinger
}

func NewHealthHandler(pinger Pinger) *HealthHandler {
	return &HealthHandler{pinger: pinger}
}

// Liveness handles GET /health/live
// @Summary      Liveness probe
// @Description  Returns 200 if the service process is running
// @Tags         Health
// @Produce      json
// @Success      200 {object} map[string]string
// @Router       /health/live [get]
func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

// Readiness handles GET /health/ready
// @Summary      Readiness probe
// @Description  Returns 200 if the service is ready to accept traffic
// @Tags         Health
// @Produce      json
// @Success      200 {object} map[string]string
// @Failure      503 {object} map[string]string
// @Router       /health/ready [get]
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	if err := h.pinger.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
