package admin

import (
	"errors"
	"net/http"

	"masterbook/internal/auth"
	"masterbook/internal/httpx"
)

type Handler struct{ service *Service }

type CreateMasterRequest struct {
	Name        string  `json:"name"`
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	Description string  `json:"description"`
	PhotoURL    *string `json:"photo_url"`
}

type CreateMasterResponse struct {
	UserID    int64  `json:"user_id"`
	ProfileID int64  `json:"profile_id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("POST /api/v1/admin/masters", authMiddleware(auth.RequireRole("admin")(http.HandlerFunc(h.CreateMaster))))
}

func (h *Handler) CreateMaster(w http.ResponseWriter, r *http.Request) {
	var req CreateMasterRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.service.CreateMaster(r.Context(), req.Name, req.Email, req.Password, req.Description, req.PhotoURL)
	switch {
	case errors.Is(err, auth.ErrValidation):
		httpx.Error(w, http.StatusBadRequest, "invalid master data")
	case errors.Is(err, auth.ErrEmailExists):
		httpx.Error(w, http.StatusConflict, "email already exists")
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "failed to create master")
	default:
		httpx.JSON(w, http.StatusCreated, CreateMasterResponse{
			UserID:    result.UserID,
			ProfileID: result.ProfileID,
			Name:      result.Name,
			Email:     result.Email,
		})
	}
}
