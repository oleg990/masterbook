package auth

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailExists        = errors.New("email already exists")
	ErrValidation         = errors.New("validation error")
	ErrUserNotFound       = errors.New("user not found")
)

type Service struct {
	users     UserRepository
	jwtSecret string
}

func NewService(users UserRepository, jwtSecret string) *Service {
	return &Service{users: users, jwtSecret: jwtSecret}
}

func (s *Service) Register(ctx context.Context, name, email, password string) (User, error) {
	return s.register(ctx, name, email, password, "client")
}

// RegisterWithRole создаёт мастера от имени администратора. Обычная регистрация
// (Register) всегда даёт роль client — сюда роль передаёт только admin-модуль.
func (s *Service) RegisterWithRole(ctx context.Context, name, email, password, role string) (User, error) {
	if role != "master" {
		return User{}, ErrValidation
	}
	return s.register(ctx, name, email, password, role)
}

func (s *Service) register(ctx context.Context, name, email, password, role string) (User, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))

	if name == "" || email == "" || len(password) < 8 || len([]byte(password)) > 72 {
		return User{}, ErrValidation
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}

	user, err := s.users.Create(ctx, name, email, string(passwordHash), role)
	if err != nil && strings.Contains(err.Error(), "duplicate key") {
		return User{}, ErrEmailExists
	}
	return user, err
}

func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return "", ErrValidation
	}

	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) || strings.Contains(err.Error(), "no rows") {
			return "", ErrInvalidCredentials
		}
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	return CreateToken(user.ID, user.Email, user.Role, s.jwtSecret)
}

func (s *Service) Me(ctx context.Context, userID int64) (User, error) {
	return s.users.FindByID(ctx, userID)
}
