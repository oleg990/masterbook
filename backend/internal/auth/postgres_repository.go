package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) UserRepository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, name, email, passwordHash, role string) (User, error) {
	var user User
	err := r.db.QueryRow(ctx, `
        INSERT INTO users (name, email, password_hash, role)
        VALUES ($1, $2, $3, $4)
        RETURNING id, name, email, password_hash, role
    `, name, email, passwordHash, role).Scan(
		&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.Role,
	)
	return user, err
}

func (r *postgresRepository) FindByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := r.db.QueryRow(ctx, `
        SELECT id, name, email, password_hash, role
        FROM users
        WHERE email = $1
    `, email).Scan(
		&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.Role,
	)
	return user, err
}

func (r *postgresRepository) FindByID(ctx context.Context, userID int64) (User, error) {
	var user User
	err := r.db.QueryRow(ctx, `
        SELECT id, name, email, password_hash, role
        FROM users
        WHERE id = $1
    `, userID).Scan(
		&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.Role,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	return user, err
}
