package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/AlbertoDePena/go-api-poc/internal/domain"
	"github.com/AlbertoDePena/go-api-poc/internal/service"
)

type GreetingHandler struct {
	svc service.GreetingService
}

func NewGreetingHandler(svc service.GreetingService) *GreetingHandler {
	return &GreetingHandler{svc: svc}
}

// CreateGreeting handles POST /api/v1/greetings
// @Summary      Create a greeting
// @Description  Create a new greeting with the given name
// @Tags         Greetings
// @Accept       json
// @Produce      json
// @Param        request body     handler.CreateGreetingRequest true "Greeting request"
// @Success      201     {object} handler.GreetingResponse
// @Failure      400 	 {object} handler.ErrorResponse
// @Failure      404 	 {object} handler.ErrorResponse
// @Failure      500     {object} handler.ErrorResponse
// @Router       /api/v1/greetings [post]
func (h *GreetingHandler) CreateGreeting(w http.ResponseWriter, r *http.Request) {
	var req CreateGreetingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	greeting, err := h.svc.CreateGreeting(r.Context(), req.ToParams())
	if err != nil {
		writeJSON(w, domainErrToHTTP(err), ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, GreetingFromDomain(greeting))
}

// GetGreeting handles GET /api/v1/greetings/{id}
// @Summary      Get a greeting by ID
// @Description  Retrieve a greeting by its unique identifier
// @Tags         Greetings
// @Produce      json
// @Param        id   path     string true "Greeting ID"
// @Success      200  {object} handler.GreetingResponse
// @Failure      400 	 {object} handler.ErrorResponse
// @Failure      404 	 {object} handler.ErrorResponse
// @Failure      500  {object} handler.ErrorResponse
// @Router       /api/v1/greetings/{id} [get]
func (h *GreetingHandler) GetGreeting(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	greeting, err := h.svc.GetGreeting(r.Context(), id)
	if err != nil {
		writeJSON(w, domainErrToHTTP(err), ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, GreetingFromDomain(greeting))
}

// ListGreetings handles GET /api/v1/greetings
// @Summary      List all greetings
// @Description  Retrieve all greetings
// @Tags         Greetings
// @Produce      json
// @Success      200 {array} handler.GreetingResponse
// @Failure      500 {object} handler.ErrorResponse
// @Router       /api/v1/greetings [get]
func (h *GreetingHandler) ListGreetings(w http.ResponseWriter, r *http.Request) {
	greetings, err := h.svc.ListGreetings(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, GreetingsFromDomain(greetings))
}

func domainErrToHTTP(err error) int {
	switch {
	case errors.Is(err, domain.ErrGreetingNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrNameRequired):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
