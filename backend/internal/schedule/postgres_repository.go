package schedule

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) Repository { return &postgresRepository{db: db} }

func (r *postgresRepository) Upsert(ctx context.Context, masterID int64, day int, startTime, endTime string) (WorkingHour, error) {
	var w WorkingHour
	err := r.db.QueryRow(ctx, `
        INSERT INTO working_hours (master_id, day_of_week, start_time, end_time)
        VALUES ($1,$2,$3,$4)
        ON CONFLICT (master_id, day_of_week)
        DO UPDATE SET start_time=EXCLUDED.start_time, end_time=EXCLUDED.end_time, updated_at=NOW()
        RETURNING id, master_id, day_of_week, start_time, end_time
    `, masterID, day, startTime, endTime).Scan(&w.ID, &w.MasterID, &w.DayOfWeek, &w.StartTime, &w.EndTime)
	return w, err
}

func (r *postgresRepository) ListByMaster(ctx context.Context, masterID int64) ([]WorkingHour, error) {
	rows, err := r.db.Query(ctx, `SELECT id, master_id, day_of_week, start_time, end_time FROM working_hours WHERE master_id=$1 ORDER BY day_of_week`, masterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]WorkingHour, 0)
	for rows.Next() {
		var w WorkingHour
		if err := rows.Scan(&w.ID, &w.MasterID, &w.DayOfWeek, &w.StartTime, &w.EndTime); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *postgresRepository) FindByMasterAndDay(ctx context.Context, masterID int64, day int) (WorkingHour, error) {
	var w WorkingHour
	err := r.db.QueryRow(ctx, `SELECT id, master_id, day_of_week, start_time, end_time FROM working_hours WHERE master_id=$1 AND day_of_week=$2`, masterID, day).Scan(&w.ID, &w.MasterID, &w.DayOfWeek, &w.StartTime, &w.EndTime)
	if errors.Is(err, pgx.ErrNoRows) {
		return WorkingHour{}, ErrNotFound
	}
	return w, err
}

func (r *postgresRepository) DeleteByMasterAndDay(ctx context.Context, masterID int64, day int) error {
	cmd, err := r.db.Exec(ctx, `DELETE FROM working_hours WHERE master_id=$1 AND day_of_week=$2`, masterID, day)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
