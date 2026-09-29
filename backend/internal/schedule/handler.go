package schedule

import (
	"errors"
	"net/http"
	"strconv"

	"masterbook/internal/auth"
	"masterbook/internal/httpx"
)

type Handler struct{ service *Service }
type WorkingHourRequest struct {
	DayOfWeek int    `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}
type WorkingHourResponse struct {
	ID        int64  `json:"id"`
	DayOfWeek int    `json:"day_of_week"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("PUT /api/v1/master/schedule", authMiddleware(http.HandlerFunc(h.SetWorkingHour)))
	mux.Handle("DELETE /api/v1/master/schedule/{day}", authMiddleware(http.HandlerFunc(h.DeleteWorkingHour)))
	mux.HandleFunc("GET /api/v1/masters/{masterID}/schedule", h.GetWorkingHours)
	mux.Handle("PUT /api/v1/admin/masters/{masterID}/schedule", authMiddleware(auth.RequireRole("admin")(http.HandlerFunc(h.AdminSetWorkingHour))))
	mux.Handle("DELETE /api/v1/admin/masters/{masterID}/schedule/{day}", authMiddleware(auth.RequireRole("admin")(http.HandlerFunc(h.AdminDeleteWorkingHour))))
}

func toResponse(w WorkingHour) WorkingHourResponse {
	return WorkingHourResponse{ID: w.ID, DayOfWeek: w.DayOfWeek, StartTime: w.StartTime.Format("15:04"), EndTime: w.EndTime.Format("15:04")}
}

func (h *Handler) SetWorkingHour(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}
	var req WorkingHourRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.SetWorkingHourAsOwner(r.Context(), userID, req.DayOfWeek, req.StartTime, req.EndTime)
	switch {
	case errors.Is(err, ErrValidation):
		httpx.Error(w, http.StatusBadRequest, "invalid working hours")
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "master profile not found")
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "failed to save working hours")
	default:
		httpx.JSON(w, http.StatusOK, toResponse(item))
	}
}
func (h *Handler) GetWorkingHours(w http.ResponseWriter, r *http.Request) {
	masterID, err := strconv.ParseInt(r.PathValue("masterID"), 10, 64)
	if err != nil || masterID <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid master id")
		return
	}
	items, err := h.service.GetWorkingHours(r.Context(), masterID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to get working hours")
		return
	}
	out := make([]WorkingHourResponse, 0, len(items))
	for _, item := range items {
		out = append(out, toResponse(item))
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (h *Handler) DeleteWorkingHour(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}
	day, err := strconv.Atoi(r.PathValue("day"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid day of week")
		return
	}
	err = h.service.DeleteWorkingHourAsOwner(r.Context(), userID, day)
	writeWorkingHourDeleteResult(w, day, err)
}

func (h *Handler) AdminSetWorkingHour(w http.ResponseWriter, r *http.Request) {
	masterID, err := strconv.ParseInt(r.PathValue("masterID"), 10, 64)
	if err != nil || masterID <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid master id")
		return
	}
	var req WorkingHourRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.SetWorkingHour(r.Context(), masterID, req.DayOfWeek, req.StartTime, req.EndTime)
	switch {
	case errors.Is(err, ErrValidation):
		httpx.Error(w, http.StatusBadRequest, "invalid working hours")
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "failed to save working hours")
	default:
		httpx.JSON(w, http.StatusOK, toResponse(item))
	}
}

func (h *Handler) AdminDeleteWorkingHour(w http.ResponseWriter, r *http.Request) {
	masterID, err := strconv.ParseInt(r.PathValue("masterID"), 10, 64)
	if err != nil || masterID <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid master id")
		return
	}
	day, err := strconv.Atoi(r.PathValue("day"))
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid day of week")
		return
	}
	err = h.service.DeleteWorkingHour(r.Context(), masterID, day)
	writeWorkingHourDeleteResult(w, day, err)
}

func writeWorkingHourDeleteResult(w http.ResponseWriter, day int, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		httpx.Error(w, http.StatusBadRequest, "invalid day of week")
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "master profile not found")
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "working hour not found")
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "failed to delete working hour")
	default:
		httpx.JSON(w, http.StatusOK, map[string]any{"day_of_week": day, "deleted": true})
	}
}
