package services

import (
	"context"
	"errors"
	"testing"
)

type fakeServiceRepo struct{}

func (fakeServiceRepo) Create(ctx context.Context, masterID int64, name, description string, price float64, duration int) (Model, error) {
	return Model{ID: 1, MasterID: masterID, Name: name, Description: description, Price: price, DurationMinutes: duration}, nil
}
func (fakeServiceRepo) ListByMaster(ctx context.Context, masterID int64) ([]Model, error) {
	return nil, nil
}
func (fakeServiceRepo) FindForMaster(ctx context.Context, serviceID, masterID int64) (Model, error) {
	return Model{}, ErrNotFound
}
func (fakeServiceRepo) Update(ctx context.Context, serviceID, masterID int64, name, description string, price float64, duration int) (Model, error) {
	return Model{ID: serviceID, MasterID: masterID, Name: name, Description: description, Price: price, DurationMinutes: duration}, nil
}
func (fakeServiceRepo) Delete(ctx context.Context, serviceID, masterID int64) error { return nil }

type fakeMasterIDProvider struct {
	masterID int64
	err      error
}

func (f fakeMasterIDProvider) FindMasterIDByUserID(ctx context.Context, userID int64) (int64, error) {
	return f.masterID, f.err
}

func TestCreateAsOwnerResolvesMasterID(t *testing.T) {
	s := NewService(fakeServiceRepo{}, fakeMasterIDProvider{masterID: 5})
	m, err := s.CreateAsOwner(context.Background(), 1, "Стрижка", "", 500, 30)
	if err != nil {
		t.Fatal(err)
	}
	if m.MasterID != 5 {
		t.Fatalf("expected masterID 5, got %d", m.MasterID)
	}
}

func TestUpdateAsOwnerRejectsWhenNoProfile(t *testing.T) {
	s := NewService(fakeServiceRepo{}, fakeMasterIDProvider{err: errors.New("no profile")})
	_, err := s.UpdateAsOwner(context.Background(), 1, 1, "Стрижка", "", 500, 30)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestUpdateRejectsInvalidPrice(t *testing.T) {
	s := NewService(fakeServiceRepo{}, fakeMasterIDProvider{masterID: 5})
	_, err := s.Update(context.Background(), 5, 1, "Стрижка", "", -1, 30)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}
