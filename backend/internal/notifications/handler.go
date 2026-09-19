package notifications

import (
	"context"
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

type NotificationResponse struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	Type      string `json:"type"`
	IsRead    bool   `json:"is_read"`
	CreatedAt string `json:"created_at"`
}

// GET /api/v1/notifications
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userIDString := auth.GetUserID(r)

	if userIDString == "" {
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
			id,
			title,
			message,
			type,
			is_read,
			created_at::text
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		`,
		userID,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get notifications",
		})
		return
	}

	defer rows.Close()

	result := make([]NotificationResponse, 0)

	for rows.Next() {
		var item NotificationResponse

		if err := rows.Scan(
			&item.ID,
			&item.Title,
			&item.Message,
			&item.Type,
			&item.IsRead,
			&item.CreatedAt,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "failed to read notifications",
			})
			return
		}

		result = append(result, item)
	}

	writeJSON(w, http.StatusOK, result)
}

// PATCH /api/v1/notifications/{id}/read
func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	userIDString := auth.GetUserID(r)

	if userIDString == "" {
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

	notificationID, err := strconv.ParseInt(
		r.PathValue("id"),
		10,
		64,
	)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid notification id",
		})
		return
	}

	var id int64

	err = h.DB.QueryRow(
		r.Context(),
		`
		UPDATE notifications
		SET is_read = TRUE
		WHERE id = $1
		  AND user_id = $2
		RETURNING id
		`,
		notificationID,
		userID,
	).Scan(&id)

	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "notification not found",
			})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to update notification",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":      id,
		"is_read": true,
	})
}

func Create(
	db *pgxpool.Pool,
	userID int64,
	title string,
	message string,
	notificationType string,
) error {
	_, err := db.Exec(
		context.Background(),
		`
		INSERT INTO notifications (
			user_id,
			title,
			message,
			type
		)
		VALUES ($1, $2, $3, $4)
		`,
		userID,
		title,
		message,
		notificationType,
	)

	return err
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}
