package appointments

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"masterbook/internal/auth"
)

type Handler struct {
	DB *pgxpool.Pool
}

type CreateAppointmentRequest struct {
	MasterID  int64  `json:"master_id"`
	ServiceID int64  `json:"service_id"`
	StartTime string `json:"start_time"`
}

type AppointmentResponse struct {
	ID          int64  `json:"id"`
	MasterID    int64  `json:"master_id"`
	ServiceID   int64  `json:"service_id"`
	MasterName  string `json:"master_name"`
	ServiceName string `json:"service_name"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Status      string `json:"status"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userIDString, ok := auth.GetUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "user is not authenticated",
		})
		return
	}

	clientID, err := strconv.ParseInt(userIDString, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "invalid user id",
		})
		return
	}

	var req CreateAppointmentRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid JSON",
		})
		return
	}

	if req.MasterID <= 0 || req.ServiceID <= 0 || req.StartTime == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "master_id, service_id and start_time are required",
		})
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "start_time must have RFC3339 format",
		})
		return
	}

	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to start transaction",
		})
		return
	}
	defer tx.Rollback(r.Context())

	// Get the service and make sure it belongs to the selected master.
	var (
		serviceMasterID int64
		durationMinutes int
	)

	err = tx.QueryRow(
		r.Context(),
		`
		SELECT master_id, duration_minutes
		FROM services
		WHERE id = $1
		`,
		req.ServiceID,
	).Scan(
		&serviceMasterID,
		&durationMinutes,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "service not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get service",
		})
		return
	}

	if serviceMasterID != req.MasterID {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "service does not belong to this master",
		})
		return
	}

	// Lock the master row while we check and create the appointment.
	// This prevents two simultaneous requests from booking the same time.
	var lockedMasterID int64

	err = tx.QueryRow(
		r.Context(),
		`
		SELECT id
		FROM master_profiles
		WHERE id = $1
		FOR UPDATE
		`,
		req.MasterID,
	).Scan(&lockedMasterID)

	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "master not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get master",
		})
		return
	}

	endTime := startTime.Add(time.Duration(durationMinutes) * time.Minute)

	// Check for overlapping active appointments.
	var existingID int64

	err = tx.QueryRow(
		r.Context(),
		`
		SELECT id
		FROM appointments
		WHERE master_id = $1
		  AND status IN ('pending', 'confirmed')
		  AND start_time < $3
		  AND end_time > $2
		LIMIT 1
		`,
		req.MasterID,
		startTime,
		endTime,
	).Scan(&existingID)

	if err == nil {
		writeJSON(w, http.StatusConflict, map[string]string{
			"error": "this time is already booked",
		})
		return
	}

	if err != pgx.ErrNoRows {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to check appointment availability",
		})
		return
	}

	var appointmentID int64

	err = tx.QueryRow(
		r.Context(),
		`
		INSERT INTO appointments (
			client_id,
			master_id,
			service_id,
			start_time,
			end_time,
			status
		)
		VALUES ($1, $2, $3, $4, $5, 'pending')
		RETURNING id
		`,
		clientID,
		req.MasterID,
		req.ServiceID,
		startTime,
		endTime,
	).Scan(&appointmentID)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to create appointment",
		})
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to save appointment",
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":         appointmentID,
		"master_id":  req.MasterID,
		"service_id": req.ServiceID,
		"start_time": startTime,
		"end_time":   endTime,
		"status":     "pending",
	})
}

func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) {
	userIDString, ok := auth.GetUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "user is not authenticated",
		})
		return
	}

	clientID, err := strconv.ParseInt(userIDString, 10, 64)
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
			a.master_id,
			a.service_id,
			u.name,
			s.name,
			a.start_time,
			a.end_time,
			a.status
		FROM appointments a
		JOIN master_profiles mp ON mp.id = a.master_id
		JOIN users u ON u.id = mp.user_id
		JOIN services s ON s.id = a.service_id
		WHERE a.client_id = $1
		ORDER BY a.start_time
		`,
		clientID,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get appointments",
		})
		return
	}
	defer rows.Close()

	result := make([]AppointmentResponse, 0)

	for rows.Next() {
		var item AppointmentResponse
		var startTime time.Time
		var endTime time.Time

		if err := rows.Scan(
			&item.ID,
			&item.MasterID,
			&item.ServiceID,
			&item.MasterName,
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

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	userIDString, ok := auth.GetUserID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "user is not authenticated",
		})
		return
	}

	clientID, err := strconv.ParseInt(userIDString, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "invalid user id",
		})
		return
	}

	appointmentID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid appointment id",
		})
		return
	}

	var status string

	err = h.DB.QueryRow(
		r.Context(),
		`
		UPDATE appointments
		SET
			status = 'cancelled',
			updated_at = NOW()
		WHERE id = $1
		  AND client_id = $2
		  AND status IN ('pending', 'confirmed')
		RETURNING status
		`,
		appointmentID,
		clientID,
	).Scan(&status)

	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "appointment not found or cannot be cancelled",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to cancel appointment",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": status,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}
