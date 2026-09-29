package masters

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) Repository { return &postgresRepository{db: db} }

func (r *postgresRepository) GetUserRole(ctx context.Context, userID int64) (string, error) {
	var role string
	err := r.db.QueryRow(ctx, `SELECT role FROM users WHERE id = $1`, userID).Scan(&role)
	return role, err
}

func (r *postgresRepository) FindByUserID(ctx context.Context, userID int64) (Profile, error) {
	var p Profile
	err := r.db.QueryRow(ctx, `
        SELECT mp.id, mp.user_id, u.name, mp.description, mp.photo_url
        FROM master_profiles mp
        JOIN users u ON u.id = mp.user_id
        WHERE mp.user_id = $1
    `, userID).Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.PhotoURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrProfileNotFound
	}
	return p, err
}

func (r *postgresRepository) CreateProfile(ctx context.Context, userID int64, description string, photoURL *string) (Profile, error) {
	var p Profile
	err := r.db.QueryRow(ctx, `
        INSERT INTO master_profiles (user_id, description, photo_url)
        VALUES ($1, $2, $3)
        RETURNING id, user_id, description, photo_url
    `, userID, description, photoURL).Scan(&p.ID, &p.UserID, &p.Description, &p.PhotoURL)
	if err != nil {
		return Profile{}, err
	}
	p.Name, err = r.userName(ctx, userID)
	return p, err
}

func (r *postgresRepository) UpdateProfile(ctx context.Context, masterID int64, description string, photoURL *string) (Profile, error) {
	var p Profile
	err := r.db.QueryRow(ctx, `
        UPDATE master_profiles SET description = $1, photo_url = $2, updated_at = NOW()
        WHERE id = $3
        RETURNING id, user_id, description, photo_url
    `, description, photoURL, masterID).Scan(&p.ID, &p.UserID, &p.Description, &p.PhotoURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrProfileNotFound
	}
	if err != nil {
		return Profile{}, err
	}
	p.Name, err = r.userName(ctx, p.UserID)
	return p, err
}

func (r *postgresRepository) UpdatePhotoURL(ctx context.Context, masterID int64, photoURL string) (Profile, error) {
	var p Profile
	err := r.db.QueryRow(ctx, `
        UPDATE master_profiles SET photo_url = $1, updated_at = NOW()
        WHERE id = $2
        RETURNING id, user_id, description, photo_url
    `, photoURL, masterID).Scan(&p.ID, &p.UserID, &p.Description, &p.PhotoURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrProfileNotFound
	}
	if err != nil {
		return Profile{}, err
	}
	p.Name, err = r.userName(ctx, p.UserID)
	return p, err
}

func (r *postgresRepository) userName(ctx context.Context, userID int64) (string, error) {
	var name string
	err := r.db.QueryRow(ctx, `SELECT name FROM users WHERE id = $1`, userID).Scan(&name)
	return name, err
}

func (r *postgresRepository) List(ctx context.Context) ([]Profile, error) {
	rows, err := r.db.Query(ctx, `
        SELECT mp.id, u.id, u.name, mp.description, mp.photo_url
        FROM master_profiles mp
        JOIN users u ON u.id = mp.user_id
        WHERE u.role = 'master'
        ORDER BY u.name
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Profile, 0)
	for rows.Next() {
		var p Profile
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.PhotoURL); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}
