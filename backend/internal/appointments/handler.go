package appointments

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"masterbook/internal/auth"
	"masterbook/internal/httpx"
)

type Handler struct{ service *Service }
type CreateAppointmentRequest struct {
	MasterID  int64  `json:"master_id"`
	ServiceID int64  `json:"service_id"`
	StartTime string `json:"start_time"`
}
type AppointmentResponse struct {
	ID        int64     `json:"id"`
	MasterID  int64     `json:"master_id"`
	ServiceID int64     `json:"service_id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"`
}
type AppointmentListItem struct {
	ID          int64     `json:"id"`
	MasterID    int64     `json:"master_id"`
	ServiceID   int64     `json:"service_id"`
	MasterName  string    `json:"master_name"`
	ServiceName string    `json:"service_name"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Status      string    `json:"status"`
}
type MasterAppointmentResponse struct {
	ID          int64  `json:"id"`
	ClientName  string `json:"client_name"`
	ClientEmail string `json:"client_email"`
	ServiceName string `json:"service_name"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Status      string `json:"status"`
}
type AvailabilitySlot struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("POST /api/v1/appointments", authMiddleware(http.HandlerFunc(h.Create)))
	mux.Handle("GET /api/v1/appointments", authMiddleware(http.HandlerFunc(h.ListMine)))
	mux.Handle("PATCH /api/v1/appointments/{id}/cancel", authMiddleware(http.HandlerFunc(h.Cancel)))
	mux.HandleFunc("GET /api/v1/masters/{masterID}/availability", h.GetAvailability)
	mux.Handle("GET /api/v1/master/appointments", authMiddleware(http.HandlerFunc(h.ListMasterAppointments)))
	mux.Handle("PATCH /api/v1/master/appointments/{id}/confirm", authMiddleware(http.HandlerFunc(h.Confirm)))
	mux.Handle("PATCH /api/v1/master/appointments/{id}/complete", authMiddleware(http.HandlerFunc(h.Complete)))
	mux.Handle("PATCH /api/v1/master/appointments/{id}/cancel", authMiddleware(http.HandlerFunc(h.CancelByMaster)))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	clientID, ok := auth.GetUserID(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid user id")
		return
	}
	var req CreateAppointmentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	start, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid start_time")
		return
	}
	a, err := h.service.Create(r.Context(), clientID, req.MasterID, req.ServiceID, start)
	switch {
	case errors.Is(err, ErrServiceNotFound):
		httpx.Error(w, http.StatusBadRequest, "service not found for this master")
	case errors.Is(err, ErrConflict):
		httpx.Error(w, http.StatusConflict, "time slot is already booked")
	case errors.Is(err, ErrOutsideWorkingHours):
		httpx.Error(w, http.StatusBadRequest, "appointment is outside working hours")
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "master not found")
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "failed to create appointment")
	default:
		httpx.JSON(w, http.StatusCreated, AppointmentResponse{ID: a.ID, MasterID: a.MasterID, ServiceID: a.ServiceID, StartTime: a.StartTime, EndTime: a.EndTime, Status: a.Status})
	}
}
func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) {
	clientID, ok := auth.GetUserID(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid user id")
		return
	}
	items, err := h.service.ListMine(r.Context(), clientID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to load appointments")
		return
	}
	out := make([]AppointmentListItem, 0, len(items))
	for _, x := range items {
		out = append(out, AppointmentListItem{ID: x.ID, MasterID: x.MasterID, ServiceID: x.ServiceID, MasterName: x.MasterName, ServiceName: x.ServiceName, StartTime: x.StartTime, EndTime: x.EndTime, Status: x.Status})
	}
	httpx.JSON(w, http.StatusOK, out)
}
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	clientID, ok := auth.GetUserID(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "invalid user id")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid appointment id")
		return
	}
	status, err := h.service.Cancel(r.Context(), id, clientID)
	if errors.Is(err, ErrNotFound) {
		httpx.Error(w, http.StatusNotFound, "appointment not found or cannot be cancelled")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to cancel appointment")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"id": id, "status": status})
}
func (h *Handler) GetAvailability(w http.ResponseWriter, r *http.Request) {
	masterID, err := strconv.ParseInt(r.PathValue("masterID"), 10, 64)
	if err != nil || masterID <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid master id")
		return
	}
	serviceID := int64(0)
	if rawServiceID := r.URL.Query().Get("service_id"); rawServiceID != "" {
		serviceID, err = strconv.ParseInt(rawServiceID, 10, 64)
		if err != nil || serviceID <= 0 {
			httpx.Error(w, http.StatusBadRequest, "invalid service id")
			return
		}
	}
	dateString := r.URL.Query().Get("date")
	if dateString == "" {
		httpx.Error(w, http.StatusBadRequest, "date is required")
		return
	}
	date, err := time.Parse("2006-01-02", dateString)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid date, expected YYYY-MM-DD")
		return
	}
	result, err := h.service.Availability(r.Context(), masterID, serviceID, date)
	if errors.Is(err, ErrServiceNotFound) {
		httpx.Error(w, http.StatusBadRequest, "service not found for this master")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to get availability")
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}
func (h *Handler) ListMasterAppointments(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}
	items, err := h.service.ListMasterAppointments(r.Context(), userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to get appointments")
		return
	}
	out := make([]MasterAppointmentResponse, 0, len(items))
	for _, x := range items {
		out = append(out, MasterAppointmentResponse{ID: x.ID, ClientName: x.ClientName, ClientEmail: x.ClientEmail, ServiceName: x.ServiceName, StartTime: x.StartTime.Format(time.RFC3339), EndTime: x.EndTime.Format(time.RFC3339), Status: x.Status})
	}
	httpx.JSON(w, http.StatusOK, out)
}
func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request)  { h.changeStatus(w, r, "confirmed") }
func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) { h.changeStatus(w, r, "completed") }
func (h *Handler) CancelByMaster(w http.ResponseWriter, r *http.Request) {
	h.changeStatus(w, r, "cancelled")
}
func (h *Handler) changeStatus(w http.ResponseWriter, r *http.Request, newStatus string) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid appointment id")
		return
	}
	status, err := h.service.ChangeStatus(r.Context(), id, userID, newStatus)
	switch {
	case errors.Is(err, ErrInvalidStatus):
		httpx.Error(w, http.StatusNotFound, "appointment not found or status change is not allowed")
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "failed to change appointment status")
	default:
		httpx.JSON(w, http.StatusOK, map[string]string{"status": status})
	}
}
