package services

import (
	"errors"
	"net/http"
	"strconv"

	"masterbook/internal/auth"
	"masterbook/internal/httpx"
)

type Handler struct{ service *Service }

type CreateServiceRequest struct {
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Price           float64 `json:"price"`
	DurationMinutes int     `json:"duration_minutes"`
}

type ServiceResponse struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Price           float64 `json:"price"`
	DurationMinutes int     `json:"duration_minutes"`
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("POST /api/v1/master/services", authMiddleware(http.HandlerFunc(h.Create)))
	mux.Handle("PUT /api/v1/master/services/{id}", authMiddleware(http.HandlerFunc(h.Update)))
	mux.Handle("DELETE /api/v1/master/services/{id}", authMiddleware(http.HandlerFunc(h.Delete)))
	mux.HandleFunc("GET /api/v1/masters/{masterID}/services", h.List)
	mux.Handle("POST /api/v1/admin/masters/{masterID}/services", authMiddleware(auth.RequireRole("admin")(http.HandlerFunc(h.AdminCreate))))
	mux.Handle("PUT /api/v1/admin/masters/{masterID}/services/{id}", authMiddleware(auth.RequireRole("admin")(http.HandlerFunc(h.AdminUpdate))))
	mux.Handle("DELETE /api/v1/admin/masters/{masterID}/services/{id}", authMiddleware(auth.RequireRole("admin")(http.HandlerFunc(h.AdminDelete))))
}

func toResponse(m Model) ServiceResponse {
	return ServiceResponse{ID: m.ID, Name: m.Name, Description: m.Description, Price: m.Price, DurationMinutes: m.DurationMinutes}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}
	var req CreateServiceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.CreateAsOwner(r.Context(), userID, req.Name, req.Description, req.Price, req.DurationMinutes)
	switch {
	case errors.Is(err, ErrValidation):
		httpx.Error(w, http.StatusBadRequest, "invalid service data")
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "master profile not found")
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "failed to create service")
	default:
		httpx.JSON(w, http.StatusCreated, toResponse(item))
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	masterID, err := strconv.ParseInt(r.PathValue("masterID"), 10, 64)
	if err != nil || masterID <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid master id")
		return
	}
	result, err := h.service.List(r.Context(), masterID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to get services")
		return
	}
	out := make([]ServiceResponse, 0, len(result))
	for _, m := range result {
		out = append(out, toResponse(m))
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid service id")
		return
	}
	var req CreateServiceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.UpdateAsOwner(r.Context(), userID, id, req.Name, req.Description, req.Price, req.DurationMinutes)
	writeServiceResult(w, item, err)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid service id")
		return
	}
	err = h.service.DeleteAsOwner(r.Context(), userID, id)
	writeServiceDeleteResult(w, id, err)
}

func (h *Handler) AdminCreate(w http.ResponseWriter, r *http.Request) {
	masterID, err := strconv.ParseInt(r.PathValue("masterID"), 10, 64)
	if err != nil || masterID <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid master id")
		return
	}
	var req CreateServiceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.Create(r.Context(), masterID, req.Name, req.Description, req.Price, req.DurationMinutes)
	switch {
	case errors.Is(err, ErrValidation):
		httpx.Error(w, http.StatusBadRequest, "invalid service data")
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "failed to create service")
	default:
		httpx.JSON(w, http.StatusCreated, toResponse(item))
	}
}

func (h *Handler) AdminUpdate(w http.ResponseWriter, r *http.Request) {
	masterID, err := strconv.ParseInt(r.PathValue("masterID"), 10, 64)
	if err != nil || masterID <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid master id")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid service id")
		return
	}
	var req CreateServiceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.Update(r.Context(), masterID, id, req.Name, req.Description, req.Price, req.DurationMinutes)
	writeServiceResult(w, item, err)
}

func (h *Handler) AdminDelete(w http.ResponseWriter, r *http.Request) {
	masterID, err := strconv.ParseInt(r.PathValue("masterID"), 10, 64)
	if err != nil || masterID <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid master id")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid service id")
		return
	}
	err = h.service.Delete(r.Context(), masterID, id)
	writeServiceDeleteResult(w, id, err)
}

func writeServiceResult(w http.ResponseWriter, item Model, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		httpx.Error(w, http.StatusBadRequest, "invalid service data")
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "master profile not found")
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "service not found")
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "failed to update service")
	default:
		httpx.JSON(w, http.StatusOK, toResponse(item))
	}
}

func writeServiceDeleteResult(w http.ResponseWriter, id int64, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "master profile not found")
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "service not found")
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "failed to delete service")
	default:
		httpx.JSON(w, http.StatusOK, map[string]any{"id": id, "deleted": true})
	}
}
