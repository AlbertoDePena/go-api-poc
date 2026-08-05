package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/AlbertoDePena/go-api-poc/internal/adapter/http/dto"
	"github.com/AlbertoDePena/go-api-poc/internal/core/domain"
	"github.com/AlbertoDePena/go-api-poc/internal/core/usecase"
)

type GreetingHandler struct {
	createGreeting usecase.CreateGreetingUseCase
	getGreeting    usecase.GetGreetingUseCase
	listGreetings  usecase.ListGreetingsUseCase
}

func NewGreetingHandler(
	create usecase.CreateGreetingUseCase,
	get usecase.GetGreetingUseCase,
	list usecase.ListGreetingsUseCase,
) *GreetingHandler {
	return &GreetingHandler{
		createGreeting: create,
		getGreeting:    get,
		listGreetings:  list,
	}
}

// CreateGreeting handles POST /api/v1/greetings
// @Summary      Create a greeting
// @Description  Create a new greeting with the given name
// @Tags         Greetings
// @Accept       json
// @Produce      json
// @Param        request body     dto.CreateGreetingRequest true "Greeting request"
// @Success      201     {object} dto.GreetingResponse
// @Failure      400 	 {object} dto.ErrorResponse
// @Failure      404 	 {object} dto.ErrorResponse
// @Failure      500     {object} dto.ErrorResponse
// @Router       /api/v1/greetings [post]
func (h *GreetingHandler) CreateGreeting(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateGreetingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, dto.ErrorResponse{Error: "invalid request body"})
		return
	}

	greeting, err := h.createGreeting.Execute(r.Context(), req.ToParams())
	if err != nil {
		writeJSON(w, domainErrToHTTP(err), dto.ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, dto.GreetingFromDomain(greeting))
}

// GetGreeting handles GET /api/v1/greetings/{id}
// @Summary      Get a greeting by ID
// @Description  Retrieve a greeting by its unique identifier
// @Tags         Greetings
// @Produce      json
// @Param        id   path     string true "Greeting ID"
// @Success      200  {object} dto.GreetingResponse
// @Failure      400 	 {object} dto.ErrorResponse
// @Failure      404 	 {object} dto.ErrorResponse
// @Failure      500  {object} dto.ErrorResponse
// @Router       /api/v1/greetings/{id} [get]
func (h *GreetingHandler) GetGreeting(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	greeting, err := h.getGreeting.Execute(r.Context(), id)
	if err != nil {
		writeJSON(w, domainErrToHTTP(err), dto.ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, dto.GreetingFromDomain(greeting))
}

// ListGreetings handles GET /api/v1/greetings
// @Summary      List all greetings
// @Description  Retrieve all greetings
// @Tags         Greetings
// @Produce      json
// @Success      200 {array} dto.GreetingResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /api/v1/greetings [get]
func (h *GreetingHandler) ListGreetings(w http.ResponseWriter, r *http.Request) {
	greetings, err := h.listGreetings.Execute(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, dto.ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, dto.GreetingsFromDomain(greetings))
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
