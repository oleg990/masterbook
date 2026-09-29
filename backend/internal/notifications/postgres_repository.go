package notifications

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) Repository { return &postgresRepository{db: db} }
func (r *postgresRepository) Create(ctx context.Context, userID int64, title, message, notificationType string) error {
	_, err := r.db.Exec(ctx, `INSERT INTO notifications(user_id,title,message,type)VALUES($1,$2,$3,$4)`, userID, title, message, notificationType)
	return err
}
func (r *postgresRepository) List(ctx context.Context, userID int64) ([]Model, error) {
	rows, err := r.db.Query(ctx, `SELECT id,user_id,title,message,type,is_read,created_at::text FROM notifications WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Model, 0)
	for rows.Next() {
		var m Model
		if err := rows.Scan(&m.ID, &m.UserID, &m.Title, &m.Message, &m.Type, &m.IsRead, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (r *postgresRepository) MarkAsRead(ctx context.Context, notificationID, userID int64) error {
	var id int64
	err := r.db.QueryRow(ctx, `UPDATE notifications SET is_read=TRUE WHERE id=$1 AND user_id=$2 RETURNING id`, notificationID, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
