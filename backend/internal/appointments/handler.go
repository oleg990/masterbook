package appointments

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"masterbook/internal/auth"
	"masterbook/internal/notifications"
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

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userIDStr := auth.GetUserID(r)

	clientID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusUnauthorized)
		return
	}

	var req CreateAppointmentRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.MasterID <= 0 || req.ServiceID <= 0 {
		http.Error(w, "master_id and service_id are required", http.StatusBadRequest)
		return
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		http.Error(w, "invalid start_time", http.StatusBadRequest)
		return
	}

	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		http.Error(w, "failed to begin transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	// Получаем мастера и блокируем его строку.
	var masterUserID int64

	err = tx.QueryRow(
		r.Context(),
		`
		SELECT user_id
		FROM master_profiles
		WHERE id = $1
		FOR UPDATE
		`,
		req.MasterID,
	).Scan(&masterUserID)

	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "master not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to find master", http.StatusInternalServerError)
		return
	}

	// Проверяем, что услуга принадлежит этому мастеру
	// и получаем её длительность.
	var durationMinutes int

	err = tx.QueryRow(
		r.Context(),
		`
		SELECT duration_minutes
		FROM services
		WHERE id = $1
		  AND master_id = $2
		`,
		req.ServiceID,
		req.MasterID,
	).Scan(&durationMinutes)

	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "service not found for this master", http.StatusBadRequest)
			return
		}

		http.Error(w, "failed to find service", http.StatusInternalServerError)
		return
	}

	endTime := startTime.Add(time.Duration(durationMinutes) * time.Minute)

	// Проверяем пересечение с существующими активными записями.
	var conflictExists bool

	err = tx.QueryRow(
		r.Context(),
		`
		SELECT EXISTS (
			SELECT 1
			FROM appointments
			WHERE master_id = $1
			  AND status IN ('pending', 'confirmed')
			  AND start_time < $3
			  AND end_time > $2
		)
		`,
		req.MasterID,
		startTime,
		endTime,
	).Scan(&conflictExists)

	if err != nil {
		http.Error(w, "failed to check appointment conflict", http.StatusInternalServerError)
		return
	}

	if conflictExists {
		http.Error(w, "time slot is already booked", http.StatusConflict)
		return
	}

	var appointmentID int64
	var status string

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
		RETURNING id, status
		`,
		clientID,
		req.MasterID,
		req.ServiceID,
		startTime,
		endTime,
	).Scan(&appointmentID, &status)

	if err != nil {
		http.Error(w, "failed to create appointment", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "failed to commit appointment", http.StatusInternalServerError)
		return
	}

	// После успешного создания записи уведомляем мастера.
	// Если уведомление не создалось, саму запись не отменяем.
	err = notifications.Create(
		h.DB,
		masterUserID,
		"Новая запись",
		"К вам поступила новая запись клиента.",
		"appointment_created",
	)

	if err != nil {
		// Пока просто игнорируем ошибку уведомления.
		// Основная запись уже успешно создана.
	}

	writeJSON(w, http.StatusCreated, AppointmentResponse{
		ID:        appointmentID,
		MasterID:  req.MasterID,
		ServiceID: req.ServiceID,
		StartTime: startTime,
		EndTime:   endTime,
		Status:    status,
	})
}

func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) {
	userIDStr := auth.GetUserID(r)

	clientID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusUnauthorized)
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
		ORDER BY a.start_time DESC
		`,
		clientID,
	)
	if err != nil {
		http.Error(w, "failed to load appointments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := make([]AppointmentListItem, 0)

	for rows.Next() {
		var item AppointmentListItem

		if err := rows.Scan(
			&item.ID,
			&item.MasterID,
			&item.ServiceID,
			&item.MasterName,
			&item.ServiceName,
			&item.StartTime,
			&item.EndTime,
			&item.Status,
		); err != nil {
			http.Error(w, "failed to read appointments", http.StatusInternalServerError)
			return
		}

		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "failed to read appointments", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	userIDStr := auth.GetUserID(r)

	clientID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusUnauthorized)
		return
	}

	idStr := r.PathValue("id")

	appointmentID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid appointment id", http.StatusBadRequest)
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
			http.Error(w, "appointment not found or cannot be cancelled", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to cancel appointment", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":     appointmentID,
		"status": status,
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		return
	}
}
