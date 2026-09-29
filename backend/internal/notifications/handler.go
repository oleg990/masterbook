package notifications

import (
	"errors"
	"net/http"
	"strconv"

	"masterbook/internal/auth"
	"masterbook/internal/httpx"
)

type Handler struct{ service *Service }
type NotificationResponse struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	Type      string `json:"type"`
	IsRead    bool   `json:"is_read"`
	CreatedAt string `json:"created_at"`
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/notifications", authMiddleware(http.HandlerFunc(h.List)))
	mux.Handle("PATCH /api/v1/notifications/{id}/read", authMiddleware(http.HandlerFunc(h.MarkAsRead)))
}

func toResponse(m Model) NotificationResponse {
	return NotificationResponse{ID: m.ID, Title: m.Title, Message: m.Message, Type: m.Type, IsRead: m.IsRead, CreatedAt: m.CreatedAt}
}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}
	items, err := h.service.List(r.Context(), userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to get notifications")
		return
	}
	out := make([]NotificationResponse, 0, len(items))
	for _, m := range items {
		out = append(out, toResponse(m))
	}
	httpx.JSON(w, http.StatusOK, out)
}
func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid notification id")
		return
	}
	err = h.service.MarkAsRead(r.Context(), id, userID)
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "notification not found")
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "failed to update notification")
	default:
		httpx.JSON(w, http.StatusOK, map[string]any{"id": id, "is_read": true})
	}
}
