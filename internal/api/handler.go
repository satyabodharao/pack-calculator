// Package api is the HTTP transport layer. It translates HTTP requests into
// service calls and service results into JSON responses. It deliberately keeps
// business logic out of the handlers — those live in the service layer.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/satyabodharao/pack-calculator/internal/calculator"
	"github.com/satyabodharao/pack-calculator/internal/service"
)

// Handler holds the dependencies the HTTP handlers need.
type Handler struct {
	svc    *service.PackService
	logger *slog.Logger
}

// NewHandler constructs an API Handler.
func NewHandler(svc *service.PackService, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{svc: svc, logger: logger}
}

// --- Request / response DTOs ------------------------------------------------

type packSizesResponse struct {
	PackSizes []int `json:"pack_sizes"`
}

type packSizesRequest struct {
	PackSizes []int `json:"pack_sizes"`
}

type calculateRequest struct {
	Items int `json:"items"`
}

type calculateResponse struct {
	Items      int               `json:"items"`
	Packs      []calculator.Pack `json:"packs"`
	TotalItems int               `json:"total_items"`
	TotalPacks int               `json:"total_packs"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// --- Handlers ---------------------------------------------------------------

// PackSizes handles GET (read current sizes) and PUT (replace sizes).
func (h *Handler) PackSizes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getPackSizes(w, r)
	case http.MethodPut:
		h.putPackSizes(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) getPackSizes(w http.ResponseWriter, r *http.Request) {
	sizes, err := h.svc.GetPackSizes(r.Context())
	if err != nil {
		h.logger.Error("get pack sizes failed", "error", err)
		writeError(w, http.StatusInternalServerError, "could not retrieve pack sizes")
		return
	}
	writeJSON(w, http.StatusOK, packSizesResponse{PackSizes: sizes})
}

func (h *Handler) putPackSizes(w http.ResponseWriter, r *http.Request) {
	var req packSizesRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	sizes, err := h.svc.UpdatePackSizes(r.Context(), req.PackSizes)
	if err != nil {
		if errors.Is(err, service.ErrInvalidPackSizes) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.logger.Error("update pack sizes failed", "error", err)
		writeError(w, http.StatusInternalServerError, "could not update pack sizes")
		return
	}
	writeJSON(w, http.StatusOK, packSizesResponse{PackSizes: sizes})
}

// Calculate handles POST: given an item count, returns the packs to ship.
func (h *Handler) Calculate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req calculateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	result, err := h.svc.Calculate(r.Context(), req.Items)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidOrder):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, calculator.ErrNoPackSizes):
			writeError(w, http.StatusBadRequest, "no pack sizes configured")
		default:
			h.logger.Error("calculate failed", "error", err)
			writeError(w, http.StatusInternalServerError, "calculation failed")
		}
		return
	}

	writeJSON(w, http.StatusOK, calculateResponse{
		Items:      req.Items,
		Packs:      result.Packs,
		TotalItems: result.TotalItems,
		TotalPacks: result.TotalPacks,
	})
}

// Health is a lightweight liveness probe for Docker/Heroku.
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// --- Helpers ----------------------------------------------------------------

// decodeJSON strictly decodes a JSON request body into v, rejecting unknown
// fields so malformed requests fail fast.
func decodeJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return errors.New("empty body")
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// writeJSON writes v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// At this point the status/headers are already written, so we can only log.
		slog.Default().Error("failed to encode response", "error", err)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}
