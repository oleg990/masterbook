package appointments

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"masterbook/internal/auth"
)

type MasterAppointmentResponse struct {
	ID          int64  `json:"id"`
	ClientName  string `json:"client_name"`
	ClientEmail string `json:"client_email"`
	ServiceName string `json:"service_name"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Status      string `json:"status"`
}

// GET /api/v1/master/appointments
func (h *Handler) ListMasterAppointments(w http.ResponseWriter, r *http.Request) {
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

	rows, err := h.DB.Query(
		r.Context(),
		`
		SELECT
			a.id,
			u.name,
			u.email,
			s.name,
			a.start_time,
			a.end_time,
			a.status
		FROM appointments a
		JOIN master_profiles mp
			ON mp.id = a.master_id
		JOIN users u
			ON u.id = a.client_id
		JOIN services s
			ON s.id = a.service_id
		WHERE mp.user_id = $1
		ORDER BY a.start_time
		`,
		userID,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get appointments",
		})
		return
	}

	defer rows.Close()

	result := make([]MasterAppointmentResponse, 0)

	for rows.Next() {
		var item MasterAppointmentResponse
		var startTime time.Time
		var endTime time.Time

		if err := rows.Scan(
			&item.ID,
			&item.ClientName,
			&item.ClientEmail,
			&item.ServiceName,
			&startTime,
			&endTime,
			&item.Status,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "failed to read appointments",
			})
			return
		}

		item.StartTime = startTime.Format(time.RFC3339)
		item.EndTime = endTime.Format(time.RFC3339)

		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to read appointments",
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// PATCH /api/v1/master/appointments/{id}/confirm
func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	h.changeStatus(
		w,
		r,
		"confirmed",
		`
		AND a.status = 'pending'
		`,
	)
}

// PATCH /api/v1/master/appointments/{id}/complete
func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	h.changeStatus(
		w,
		r,
		"completed",
		`
		AND a.status = 'confirmed'
		`,
	)
}

// PATCH /api/v1/master/appointments/{id}/cancel
func (h *Handler) CancelByMaster(w http.ResponseWriter, r *http.Request) {
	h.changeStatus(
		w,
		r,
		"cancelled",
		`
		AND a.status IN ('pending', 'confirmed')
		`,
	)
}

func (h *Handler) changeStatus(
	w http.ResponseWriter,
	r *http.Request,
	newStatus string,
	statusCondition string,
) {
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

	appointmentID, err := strconv.ParseInt(
		r.PathValue("id"),
		10,
		64,
	)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid appointment id",
		})
		return
	}

	query := `
		UPDATE appointments a
		SET
			status = $1,
			updated_at = NOW()
		FROM master_profiles mp
		WHERE a.id = $2
		  AND a.master_id = mp.id
		  AND mp.user_id = $3
		` + statusCondition + `
		RETURNING a.status
	`

	var status string

	err = h.DB.QueryRow(
		r.Context(),
		query,
		newStatus,
		appointmentID,
		userID,
	).Scan(&status)

	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "appointment not found or status change is not allowed",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to change appointment status",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": status,
	})
}