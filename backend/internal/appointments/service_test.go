package appointments

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeAppointmentRepo struct{}

func (fakeAppointmentRepo) Create(context.Context, int64, int64, int64, time.Time, time.Time) (Appointment, int64, error) {
	return Appointment{ID: 1, MasterID: 1, ServiceID: 1, StartTime: time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC), EndTime: time.Date(2026, 9, 28, 11, 30, 0, 0, time.UTC), Status: "pending"}, 2, nil
}
func (fakeAppointmentRepo) ListByClient(context.Context, int64) ([]ListItem, error) { return nil, nil }
func (fakeAppointmentRepo) CancelByClient(context.Context, int64, int64) (string, error) {
	return "cancelled", nil
}
func (fakeAppointmentRepo) ListByMasterUser(context.Context, int64) ([]MasterListItem, error) {
	return nil, nil
}
func (fakeAppointmentRepo) ChangeStatus(context.Context, int64, int64, string) (string, int64, error) {
	return "confirmed", 3, nil
}
func (fakeAppointmentRepo) GetBusyIntervals(context.Context, int64, time.Time, time.Time) ([]BusyInterval, error) {
	return nil, nil
}
func (fakeAppointmentRepo) GetWorkingInterval(context.Context, int64, int) (time.Time, time.Time, error) {
	return time.Date(0, 1, 1, 9, 0, 0, 0, time.UTC), time.Date(0, 1, 1, 18, 0, 0, 0, time.UTC), nil
}
func (fakeAppointmentRepo) GetServiceDuration(context.Context, int64, int64) (int, error) {
	return 90, nil
}

type fakeNotifier struct{}

func (fakeNotifier) Create(context.Context, int64, string, string, string) error { return nil }

func TestCreateRejectsAppointmentOutsideWorkingHours(t *testing.T) {
	s := NewService(fakeAppointmentRepo{}, fakeNotifier{})
	_, err := s.Create(context.Background(), 1, 1, 1, time.Date(2026, 9, 28, 17, 0, 0, 0, time.UTC))
	if !errors.Is(err, ErrOutsideWorkingHours) {
		t.Fatalf("expected ErrOutsideWorkingHours, got %v", err)
	}
}

func TestCreateCalculatesEndFromServiceDuration(t *testing.T) {
	s := NewService(fakeAppointmentRepo{}, fakeNotifier{})
	a, err := s.Create(context.Background(), 1, 1, 1, time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 9, 28, 11, 30, 0, 0, time.UTC)
	if !a.EndTime.Equal(want) {
		t.Fatalf("unexpected end time: got %v want %v", a.EndTime, want)
	}
}
