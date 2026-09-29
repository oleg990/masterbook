package masters

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (r *postgresRepository) FindMasterIDByUserID(ctx context.Context, userID int64) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `SELECT id FROM master_profiles WHERE user_id = $1`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrProfileNotFound
	}
	return id, err
}
