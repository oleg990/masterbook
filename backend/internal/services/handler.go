package services

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"masterbook/internal/auth"
)

type Handler struct {
	DB *pgxpool.Pool
}

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

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
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

	var masterProfileID int64

	err = h.DB.QueryRow(
		r.Context(),
		`
		SELECT id
		FROM master_profiles
		WHERE user_id = $1
		`,
		userID,
	).Scan(&masterProfileID)

	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error": "master profile not found",
		})
		return
	}

	var req CreateServiceRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid JSON",
		})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "service name is required",
		})
		return
	}

	if req.Price < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "price cannot be negative",
		})
		return
	}

	if req.DurationMinutes <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "duration must be greater than 0",
		})
		return
	}

	var response ServiceResponse

	err = h.DB.QueryRow(
		r.Context(),
		`
		INSERT INTO services (
			master_id,
			name,
			description,
			price,
			duration_minutes
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, description, price, duration_minutes
		`,
		masterProfileID,
		req.Name,
		req.Description,
		req.Price,
		req.DurationMinutes,
	).Scan(
		&response.ID,
		&response.Name,
		&response.Description,
		&response.Price,
		&response.DurationMinutes,
	)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to create service",
		})
		return
	}

	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	masterID := r.PathValue("masterID")

	rows, err := h.DB.Query(
		r.Context(),
		`
		SELECT
			id,
			name,
			description,
			price,
			duration_minutes
		FROM services
		WHERE master_id = $1
		ORDER BY id
		`,
		masterID,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get services",
		})
		return
	}
	defer rows.Close()

	var result []ServiceResponse

	for rows.Next() {
		var service ServiceResponse

		if err := rows.Scan(
			&service.ID,
			&service.Name,
			&service.Description,
			&service.Price,
			&service.DurationMinutes,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "failed to read services",
			})
			return
		}

		result = append(result, service)
	}

	if result == nil {
		result = []ServiceResponse{}
	}

	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}
