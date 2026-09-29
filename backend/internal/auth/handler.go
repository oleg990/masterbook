package auth

import (
	"errors"
	"net/http"

	"masterbook/internal/httpx"
)

type Handler struct {
	service *Service
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterResponse struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	mux.HandleFunc("POST /api/v1/auth/register", h.Register)
	mux.HandleFunc("POST /api/v1/auth/login", h.Login)
	mux.Handle("GET /api/v1/me", authMiddleware(http.HandlerFunc(h.Me)))
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.service.Register(r.Context(), req.Name, req.Email, req.Password)
	switch {
	case errors.Is(err, ErrValidation):
		httpx.Error(w, http.StatusBadRequest, "invalid registration data")
	case errors.Is(err, ErrEmailExists):
		httpx.Error(w, http.StatusConflict, "email already exists")
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "failed to create user")
	default:
		httpx.JSON(w, http.StatusCreated, RegisterResponse{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Role:  user.Role,
		})
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	token, err := h.service.Login(r.Context(), req.Email, req.Password)
	switch {
	case errors.Is(err, ErrValidation):
		httpx.Error(w, http.StatusBadRequest, "email and password are required")
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Error(w, http.StatusUnauthorized, "invalid email or password")
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "failed to create token")
	default:
		httpx.JSON(w, http.StatusOK, LoginResponse{AccessToken: token, TokenType: "Bearer"})
	}
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}

	user, err := h.service.Me(r.Context(), userID)
	if errors.Is(err, ErrUserNotFound) {
		httpx.Error(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to get user")
		return
	}

	httpx.JSON(w, http.StatusOK, RegisterResponse{
		ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role,
	})
}
