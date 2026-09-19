package masters

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"masterbook/internal/auth"
)

type Handler struct {
	DB *pgxpool.Pool
}

type CreateProfileRequest struct {
	Description string `json:"description"`
	PhotoURL    string `json:"photo_url"`
}

type ProfileResponse struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PhotoURL    string `json:"photo_url,omitempty"`
}

func (h *Handler) CreateProfile(w http.ResponseWriter, r *http.Request) {
	userIDString, ok := auth.GetUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "user is not authenticated",
		})
		return
	}

	userID, err := strconv.ParseInt(userIDString, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "invalid user id",
		})
		return
	}

	var role string

	err = h.DB.QueryRow(
		r.Context(),
		`SELECT role FROM users WHERE id = $1`,
		userID,
	).Scan(&role)

	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "user not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get user",
		})
		return
	}

	if role != "master" {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error": "only master can create a master profile",
		})
		return
	}

	var req CreateProfileRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid JSON",
		})
		return
	}

	var response ProfileResponse

	err = h.DB.QueryRow(
		r.Context(),
		`
		INSERT INTO master_profiles (
			user_id,
			description,
			photo_url
		)
		VALUES ($1, $2, $3)
		RETURNING id
		`,
		userID,
		req.Description,
		req.PhotoURL,
	).Scan(&response.ID)

	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "master profile already exists",
		})
		return
	}

	response.UserID = userID

	err = h.DB.QueryRow(
		r.Context(),
		`
		SELECT
			u.name,
			mp.description,
			mp.photo_url
		FROM master_profiles mp
		JOIN users u ON u.id = mp.user_id
		WHERE mp.id = $1
		`,
		response.ID,
	).Scan(
		&response.Name,
		&response.Description,
		&response.PhotoURL,
	)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get created profile",
		})
		return
	}

	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userIDString, ok := auth.GetUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "user is not authenticated",
		})
		return
	}

	userID, err := strconv.ParseInt(userIDString, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "invalid user id",
		})
		return
	}

	var response ProfileResponse

	err = h.DB.QueryRow(
		r.Context(),
		`
		SELECT
			mp.id,
			mp.user_id,
			u.name,
			mp.description,
			mp.photo_url
		FROM master_profiles mp
		JOIN users u ON u.id = mp.user_id
		WHERE mp.user_id = $1
		`,
		userID,
	).Scan(
		&response.ID,
		&response.UserID,
		&response.Name,
		&response.Description,
		&response.PhotoURL,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "master profile not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get master profile",
		})
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}
