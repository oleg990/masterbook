package masters

import (
	"context"
	"errors"
)

type Profile struct {
	ID          int64
	UserID      int64
	Name        string
	Description string
	PhotoURL    *string
}

var ErrProfileNotFound = errors.New("master profile not found")

type Repository interface {
	FindByUserID(ctx context.Context, userID int64) (Profile, error)
	GetUserRole(ctx context.Context, userID int64) (string, error)
	CreateProfile(ctx context.Context, userID int64, description string, photoURL *string) (Profile, error)
	UpdateProfile(ctx context.Context, masterID int64, description string, photoURL *string) (Profile, error)
	UpdatePhotoURL(ctx context.Context, masterID int64, photoURL string) (Profile, error)
	List(ctx context.Context) ([]Profile, error)
	FindMasterIDByUserID(ctx context.Context, userID int64) (int64, error)
}
