package notifications

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("notification not found")

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }
func (s *Service) Create(ctx context.Context, userID int64, title, message, notificationType string) error {
	return s.repo.Create(ctx, userID, title, message, notificationType)
}
func (s *Service) List(ctx context.Context, userID int64) ([]Model, error) {
	return s.repo.List(ctx, userID)
}
func (s *Service) MarkAsRead(ctx context.Context, notificationID, userID int64) error {
	return s.repo.MarkAsRead(ctx, notificationID, userID)
}
