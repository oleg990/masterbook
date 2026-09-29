package schedule

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeScheduleRepo struct{}

func (fakeScheduleRepo) Upsert(ctx context.Context, masterID int64, day int, startTime, endTime string) (WorkingHour, error) {
	st, _ := time.Parse("15:04", startTime)
	et, _ := time.Parse("15:04", endTime)
	return WorkingHour{ID: 1, MasterID: masterID, DayOfWeek: day, StartTime: st, EndTime: et}, nil
}
func (fakeScheduleRepo) ListByMaster(ctx context.Context, masterID int64) ([]WorkingHour, error) {
	return nil, nil
}
func (fakeScheduleRepo) FindByMasterAndDay(ctx context.Context, masterID int64, day int) (WorkingHour, error) {
	return WorkingHour{}, ErrNotFound
}
func (fakeScheduleRepo) DeleteByMasterAndDay(ctx context.Context, masterID int64, day int) error {
	return nil
}

type fakeMasterIDProvider struct {
	masterID int64
	err      error
}

func (f fakeMasterIDProvider) FindMasterIDByUserID(ctx context.Context, userID int64) (int64, error) {
	return f.masterID, f.err
}

func TestSetWorkingHourAsOwnerResolvesMasterID(t *testing.T) {
	s := NewService(fakeScheduleRepo{}, fakeMasterIDProvider{masterID: 5})
	w, err := s.SetWorkingHourAsOwner(context.Background(), 1, 1, "09:00", "18:00")
	if err != nil {
		t.Fatal(err)
	}
	if w.MasterID != 5 {
		t.Fatalf("expected masterID 5, got %d", w.MasterID)
	}
}

func TestSetWorkingHourAsOwnerRejectsWhenNoProfile(t *testing.T) {
	s := NewService(fakeScheduleRepo{}, fakeMasterIDProvider{err: errors.New("no profile")})
	_, err := s.SetWorkingHourAsOwner(context.Background(), 1, 1, "09:00", "18:00")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestSetWorkingHourRejectsInvalidDay(t *testing.T) {
	s := NewService(fakeScheduleRepo{}, fakeMasterIDProvider{masterID: 5})
	_, err := s.SetWorkingHour(context.Background(), 5, 8, "09:00", "18:00")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}
