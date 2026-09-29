package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) Repository { return &postgresRepository{db: db} }

func (r *postgresRepository) Create(ctx context.Context, masterID int64, name, description string, price float64, duration int) (Model, error) {
	var m Model
	err := r.db.QueryRow(ctx, `
        INSERT INTO services (master_id, name, description, price, duration_minutes)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, master_id, name, description, price, duration_minutes
    `, masterID, name, description, price, duration).Scan(&m.ID, &m.MasterID, &m.Name, &m.Description, &m.Price, &m.DurationMinutes)
	return m, err
}

func (r *postgresRepository) ListByMaster(ctx context.Context, masterID int64) ([]Model, error) {
	rows, err := r.db.Query(ctx, `
        SELECT id, master_id, name, description, price, duration_minutes
        FROM services WHERE master_id = $1 ORDER BY id
    `, masterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Model, 0)
	for rows.Next() {
		var m Model
		if err := rows.Scan(&m.ID, &m.MasterID, &m.Name, &m.Description, &m.Price, &m.DurationMinutes); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *postgresRepository) FindForMaster(ctx context.Context, serviceID, masterID int64) (Model, error) {
	var m Model
	err := r.db.QueryRow(ctx, `
        SELECT id, master_id, name, description, price, duration_minutes
        FROM services WHERE id = $1 AND master_id = $2
    `, serviceID, masterID).Scan(&m.ID, &m.MasterID, &m.Name, &m.Description, &m.Price, &m.DurationMinutes)
	if errors.Is(err, pgx.ErrNoRows) {
		return Model{}, ErrNotFound
	}
	return m, err
}

func (r *postgresRepository) Update(ctx context.Context, serviceID, masterID int64, name, description string, price float64, duration int) (Model, error) {
	var m Model
	err := r.db.QueryRow(ctx, `
        UPDATE services SET name = $1, description = $2, price = $3, duration_minutes = $4
        WHERE id = $5 AND master_id = $6
        RETURNING id, master_id, name, description, price, duration_minutes
    `, name, description, price, duration, serviceID, masterID).Scan(&m.ID, &m.MasterID, &m.Name, &m.Description, &m.Price, &m.DurationMinutes)
	if errors.Is(err, pgx.ErrNoRows) {
		return Model{}, ErrNotFound
	}
	return m, err
}

func (r *postgresRepository) Delete(ctx context.Context, serviceID, masterID int64) error {
	cmd, err := r.db.Exec(ctx, `DELETE FROM services WHERE id = $1 AND master_id = $2`, serviceID, masterID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
