package schedule

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"masterbook/internal/auth"
)

type Handler struct {
	DB *pgxpool.Pool
}

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

func (h *Handler) SetWorkingHour(w http.ResponseWriter, r *http.Request) {
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

	var req WorkingHourRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid JSON",
		})
		return
	}

	if req.DayOfWeek < 1 || req.DayOfWeek > 7 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "day_of_week must be between 1 and 7",
		})
		return
	}

	startTime, err := time.Parse("15:04", req.StartTime)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "start_time must have format HH:MM",
		})
		return
	}

	endTime, err := time.Parse("15:04", req.EndTime)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "end_time must have format HH:MM",
		})
		return
	}

	if !startTime.Before(endTime) {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "start_time must be before end_time",
		})
		return
	}

	var response WorkingHourResponse

	err = h.DB.QueryRow(
		r.Context(),
		`
		INSERT INTO working_hours (
			master_id,
			day_of_week,
			start_time,
			end_time
		)
		SELECT
			mp.id,
			$2,
			$3,
			$4
		FROM master_profiles mp
		WHERE mp.user_id = $1
		ON CONFLICT (master_id, day_of_week)
		DO UPDATE SET
			start_time = EXCLUDED.start_time,
			end_time = EXCLUDED.end_time,
			updated_at = NOW()
		RETURNING
			id,
			day_of_week,
			start_time::text,
			end_time::text
		`,
		userID,
		req.DayOfWeek,
		req.StartTime,
		req.EndTime,
	).Scan(
		&response.ID,
		&response.DayOfWeek,
		&response.StartTime,
		&response.EndTime,
	)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to save working hours",
		})
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) GetWorkingHours(w http.ResponseWriter, r *http.Request) {
	masterID := r.PathValue("masterID")

	rows, err := h.DB.Query(
		r.Context(),
		`
		SELECT
			id,
			day_of_week,
			start_time::text,
			end_time::text
		FROM working_hours
		WHERE master_id = $1
		ORDER BY day_of_week
		`,
		masterID,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get working hours",
		})
		return
	}
	defer rows.Close()

	var result []WorkingHourResponse

	for rows.Next() {
		var item WorkingHourResponse

		if err := rows.Scan(
			&item.ID,
			&item.DayOfWeek,
			&item.StartTime,
			&item.EndTime,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "failed to read working hours",
			})
			return
		}

		result = append(result, item)
	}

	if result == nil {
		result = []WorkingHourResponse{}
	}

	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}
