package masters

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var ErrForbidden = errors.New("forbidden")

type Service struct {
	repo       Repository
	uploadsDir string
}

func NewService(repo Repository, uploadsDir string) *Service {
	return &Service{repo: repo, uploadsDir: uploadsDir}
}

func (s *Service) CreateProfile(ctx context.Context, userID int64, description string, photoURL *string) (Profile, error) {
	role, err := s.repo.GetUserRole(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	if role != "master" {
		return Profile{}, ErrForbidden
	}
	return s.repo.CreateProfile(ctx, userID, description, photoURL)
}

func (s *Service) GetProfile(ctx context.Context, userID int64) (Profile, error) {
	return s.repo.FindByUserID(ctx, userID)
}

func (s *Service) List(ctx context.Context) ([]Profile, error) {
	return s.repo.List(ctx)
}

func (s *Service) UpdateProfile(ctx context.Context, masterID int64, description string, photoURL *string) (Profile, error) {
	return s.repo.UpdateProfile(ctx, masterID, description, photoURL)
}

func (s *Service) UpdateProfileAsOwner(ctx context.Context, userID int64, description string, photoURL *string) (Profile, error) {
	masterID, err := s.repo.FindMasterIDByUserID(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	return s.UpdateProfile(ctx, masterID, description, photoURL)
}

// UploadAvatar сохраняет файл на диск и обновляет photo_url профиля.
func (s *Service) UploadAvatar(ctx context.Context, masterID int64, ext string, data []byte) (Profile, error) {
	dir := filepath.Join(s.uploadsDir, "avatars")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Profile{}, fmt.Errorf("create uploads dir: %w", err)
	}
	filename := fmt.Sprintf("%d-%d.%s", masterID, time.Now().UnixNano(), ext)
	if err := os.WriteFile(filepath.Join(dir, filename), data, 0o644); err != nil {
		return Profile{}, fmt.Errorf("write avatar file: %w", err)
	}
	return s.repo.UpdatePhotoURL(ctx, masterID, "/uploads/avatars/"+filename)
}

func (s *Service) UploadAvatarAsOwner(ctx context.Context, userID int64, ext string, data []byte) (Profile, error) {
	masterID, err := s.repo.FindMasterIDByUserID(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	return s.UploadAvatar(ctx, masterID, ext, data)
}
