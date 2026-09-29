package schedule

import (
	"context"
	"errors"
	"time"
)

var ErrValidation = errors.New("validation error")
var ErrForbidden = errors.New("forbidden")

type MasterIDProvider interface {
	FindMasterIDByUserID(ctx context.Context, userID int64) (int64, error)
}
type Service struct {
	repo   Repository
	master MasterIDProvider
}

func NewService(repo Repository, master MasterIDProvider) *Service {
	return &Service{repo: repo, master: master}
}

func (s *Service) SetWorkingHour(ctx context.Context, masterID int64, day int, startTime, endTime string) (WorkingHour, error) {
	if day < 1 || day > 7 {
		return WorkingHour{}, ErrValidation
	}
	st, err := time.Parse("15:04", startTime)
	if err != nil {
		return WorkingHour{}, ErrValidation
	}
	et, err := time.Parse("15:04", endTime)
	if err != nil || !st.Before(et) {
		return WorkingHour{}, ErrValidation
	}
	return s.repo.Upsert(ctx, masterID, day, startTime, endTime)
}
func (s *Service) SetWorkingHourAsOwner(ctx context.Context, userID int64, day int, startTime, endTime string) (WorkingHour, error) {
	masterID, err := s.master.FindMasterIDByUserID(ctx, userID)
	if err != nil {
		return WorkingHour{}, ErrForbidden
	}
	return s.SetWorkingHour(ctx, masterID, day, startTime, endTime)
}
func (s *Service) GetWorkingHours(ctx context.Context, masterID int64) ([]WorkingHour, error) {
	return s.repo.ListByMaster(ctx, masterID)
}
func (s *Service) GetWorkingHour(ctx context.Context, masterID, day int) (WorkingHour, error) {
	return s.repo.FindByMasterAndDay(ctx, int64(masterID), day)
}
func (s *Service) DeleteWorkingHour(ctx context.Context, masterID int64, day int) error {
	if day < 1 || day > 7 {
		return ErrValidation
	}
	return s.repo.DeleteByMasterAndDay(ctx, masterID, day)
}
func (s *Service) DeleteWorkingHourAsOwner(ctx context.Context, userID int64, day int) error {
	masterID, err := s.master.FindMasterIDByUserID(ctx, userID)
	if err != nil {
		return ErrForbidden
	}
	return s.DeleteWorkingHour(ctx, masterID, day)
}
