package masters

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"masterbook/internal/auth"
	"masterbook/internal/httpx"
)

const maxAvatarSize = 3 << 20 // 3 MB

var allowedAvatarExt = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

type Handler struct{ service *Service }

type ProfileRequest struct {
	Description string  `json:"description"`
	PhotoURL    *string `json:"photo_url"`
}

type ProfileResponse struct {
	ID          int64   `json:"id"`
	UserID      int64   `json:"user_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	PhotoURL    *string `json:"photo_url"`
}

type MasterListItem struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	PhotoURL    *string `json:"photo_url"`
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("POST /api/v1/master/profile", authMiddleware(http.HandlerFunc(h.CreateProfile)))
	mux.Handle("GET /api/v1/master/profile", authMiddleware(http.HandlerFunc(h.GetProfile)))
	mux.Handle("PUT /api/v1/master/profile", authMiddleware(http.HandlerFunc(h.UpdateProfile)))
	mux.Handle("POST /api/v1/master/avatar", authMiddleware(http.HandlerFunc(h.UploadAvatar)))
	mux.HandleFunc("GET /api/v1/masters", h.List)
	mux.Handle("PUT /api/v1/admin/masters/{masterID}/profile", authMiddleware(auth.RequireRole("admin")(http.HandlerFunc(h.AdminUpdateProfile))))
}

func toResponse(p Profile) ProfileResponse {
	return ProfileResponse{ID: p.ID, UserID: p.UserID, Name: p.Name, Description: p.Description, PhotoURL: p.PhotoURL}
}

func (h *Handler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}
	var req ProfileRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	profile, err := h.service.CreateProfile(r.Context(), userID, req.Description, req.PhotoURL)
	switch {
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "only master can create profile")
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "failed to create master profile")
	default:
		httpx.JSON(w, http.StatusCreated, toResponse(profile))
	}
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}
	profile, err := h.service.GetProfile(r.Context(), userID)
	switch {
	case errors.Is(err, ErrProfileNotFound):
		httpx.Error(w, http.StatusNotFound, "master profile not found")
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "failed to get master profile")
	default:
		httpx.JSON(w, http.StatusOK, toResponse(profile))
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.service.List(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to get masters")
		return
	}
	result := make([]MasterListItem, 0, len(profiles))
	for _, p := range profiles {
		result = append(result, MasterListItem{ID: p.ID, Name: p.Name, Description: p.Description, PhotoURL: p.PhotoURL})
	}
	httpx.JSON(w, http.StatusOK, result)
}

func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}
	var req ProfileRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	profile, err := h.service.UpdateProfileAsOwner(r.Context(), userID, req.Description, req.PhotoURL)
	switch {
	case errors.Is(err, ErrProfileNotFound):
		httpx.Error(w, http.StatusNotFound, "master profile not found")
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "failed to update master profile")
	default:
		httpx.JSON(w, http.StatusOK, toResponse(profile))
	}
}

func (h *Handler) AdminUpdateProfile(w http.ResponseWriter, r *http.Request) {
	masterID, err := strconv.ParseInt(r.PathValue("masterID"), 10, 64)
	if err != nil || masterID <= 0 {
		httpx.Error(w, http.StatusBadRequest, "invalid master id")
		return
	}
	var req ProfileRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	profile, err := h.service.UpdateProfile(r.Context(), masterID, req.Description, req.PhotoURL)
	switch {
	case errors.Is(err, ErrProfileNotFound):
		httpx.Error(w, http.StatusNotFound, "master profile not found")
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "failed to update master profile")
	default:
		httpx.JSON(w, http.StatusOK, toResponse(profile))
	}
}

func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r)
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "user is not authenticated")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarSize)
	if err := r.ParseMultipartForm(maxAvatarSize); err != nil {
		httpx.Error(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}
	file, _, err := r.FormFile("avatar")
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "avatar file is required")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "failed to read avatar file")
		return
	}
	ext, ok := allowedAvatarExt[http.DetectContentType(data)]
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "unsupported image type, use jpeg/png/webp")
		return
	}
	profile, err := h.service.UploadAvatarAsOwner(r.Context(), userID, ext, data)
	switch {
	case errors.Is(err, ErrProfileNotFound):
		httpx.Error(w, http.StatusNotFound, "master profile not found")
	case err != nil:
		httpx.Error(w, http.StatusInternalServerError, "failed to upload avatar")
	default:
		httpx.JSON(w, http.StatusOK, toResponse(profile))
	}
}
