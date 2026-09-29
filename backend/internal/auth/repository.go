package auth

import "context"

type User struct {
	ID           int64
	Name         string
	Email        string
	PasswordHash string
	Role         string
}

type UserRepository interface {
	Create(ctx context.Context, name, email, passwordHash, role string) (User, error)
	FindByEmail(ctx context.Context, email string) (User, error)
	FindByID(ctx context.Context, userID int64) (User, error)
}
