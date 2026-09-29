package services

import (
	"context"
	"errors"
)

type Model struct {
	ID              int64
	MasterID        int64
	Name            string
	Description     string
	Price           float64
	DurationMinutes int
}

var ErrNotFound = errors.New("service not found")

type Repository interface {
	Create(ctx context.Context, masterID int64, name, description string, price float64, duration int) (Model, error)
	ListByMaster(ctx context.Context, masterID int64) ([]Model, error)
	FindForMaster(ctx context.Context, serviceID, masterID int64) (Model, error)
	Update(ctx context.Context, serviceID, masterID int64, name, description string, price float64, duration int) (Model, error)
	Delete(ctx context.Context, serviceID, masterID int64) error
}
