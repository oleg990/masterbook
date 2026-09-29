package admin

import (
	"context"

	"masterbook/internal/auth"
	"masterbook/internal/masters"
)

// MasterCreator — узкий интерфейс поверх auth.Service, нужен только метод
// создания пользователя с заданной ролью.
type MasterCreator interface {
	RegisterWithRole(ctx context.Context, name, email, password, role string) (auth.User, error)
}

// ProfileCreator — узкий интерфейс поверх masters.Service.
type ProfileCreator interface {
	CreateProfile(ctx context.Context, userID int64, description string, photoURL *string) (masters.Profile, error)
}

type CreatedMaster struct {
	UserID    int64
	ProfileID int64
	Name      string
	Email     string
}

type Service struct {
	users    MasterCreator
	profiles ProfileCreator
}

func NewService(users MasterCreator, profiles ProfileCreator) *Service {
	return &Service{users: users, profiles: profiles}
}

// CreateMaster заводит нового мастера: сначала пользователя с ролью master
// (логин/пароль задаёт админ), затем сразу его профиль.
func (s *Service) CreateMaster(ctx context.Context, name, email, password, description string, photoURL *string) (CreatedMaster, error) {
	user, err := s.users.RegisterWithRole(ctx, name, email, password, "master")
	if err != nil {
		return CreatedMaster{}, err
	}
	profile, err := s.profiles.CreateProfile(ctx, user.ID, description, photoURL)
	if err != nil {
		return CreatedMaster{}, err
	}
	return CreatedMaster{UserID: user.ID, ProfileID: profile.ID, Name: user.Name, Email: user.Email}, nil
}
