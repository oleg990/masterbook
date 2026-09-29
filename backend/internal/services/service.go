package services

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrValidation = errors.New("validation error")
	ErrForbidden  = errors.New("forbidden")
)

// masterIDLookup keeps the service independent from the masters package.
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

func (s *Service) Create(ctx context.Context, masterID int64, name, description string, price float64, duration int) (Model, error) {
	name = strings.TrimSpace(name)
	if name == "" || price < 0 || duration <= 0 {
		return Model{}, ErrValidation
	}
	return s.repo.Create(ctx, masterID, name, description, price, duration)
}

func (s *Service) CreateAsOwner(ctx context.Context, userID int64, name, description string, price float64, duration int) (Model, error) {
	masterID, err := s.master.FindMasterIDByUserID(ctx, userID)
	if err != nil {
		return Model{}, ErrForbidden
	}
	return s.Create(ctx, masterID, name, description, price, duration)
}

func (s *Service) List(ctx context.Context, masterID int64) ([]Model, error) {
	return s.repo.ListByMaster(ctx, masterID)
}

func (s *Service) Update(ctx context.Context, masterID, serviceID int64, name, description string, price float64, duration int) (Model, error) {
	name = strings.TrimSpace(name)
	if name == "" || price < 0 || duration <= 0 {
		return Model{}, ErrValidation
	}
	return s.repo.Update(ctx, serviceID, masterID, name, description, price, duration)
}

func (s *Service) UpdateAsOwner(ctx context.Context, userID, serviceID int64, name, description string, price float64, duration int) (Model, error) {
	masterID, err := s.master.FindMasterIDByUserID(ctx, userID)
	if err != nil {
		return Model{}, ErrForbidden
	}
	return s.Update(ctx, masterID, serviceID, name, description, price, duration)
}

func (s *Service) Delete(ctx context.Context, masterID, serviceID int64) error {
	return s.repo.Delete(ctx, serviceID, masterID)
}

func (s *Service) DeleteAsOwner(ctx context.Context, userID, serviceID int64) error {
	masterID, err := s.master.FindMasterIDByUserID(ctx, userID)
	if err != nil {
		return ErrForbidden
	}
	return s.Delete(ctx, masterID, serviceID)
}
