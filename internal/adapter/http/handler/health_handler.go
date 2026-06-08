package handler

import "net/http"

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
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
// @Router       /health/ready [get]
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
