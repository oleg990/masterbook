package appointments

import (
	"context"
	"errors"
	"time"
)

type Appointment struct {
	ID        int64
	MasterID  int64
	ServiceID int64
	StartTime time.Time
	EndTime   time.Time
	Status    string
}
type ListItem struct {
	ID          int64
	MasterID    int64
	ServiceID   int64
	MasterName  string
	ServiceName string
	StartTime   time.Time
	EndTime     time.Time
	Status      string
}
type MasterListItem struct {
	ID          int64
	ClientName  string
	ClientEmail string
	ServiceName string
	StartTime   time.Time
	EndTime     time.Time
	Status      string
}

type BusyInterval struct {
	Start time.Time
	End   time.Time
}

var (
	ErrNotFound            = errors.New("appointment not found")
	ErrConflict            = errors.New("time slot is already booked")
	ErrInvalidStatus       = errors.New("appointment status change is not allowed")
	ErrServiceNotFound     = errors.New("service not found for this master")
	ErrOutsideWorkingHours = errors.New("appointment is outside working hours")
)

type Repository interface {
	Create(ctx context.Context, clientID, masterID, serviceID int64, startTime, endTime time.Time) (Appointment, int64, error)
	ListByClient(ctx context.Context, clientID int64) ([]ListItem, error)
	CancelByClient(ctx context.Context, appointmentID, clientID int64) (string, error)
	ListByMasterUser(ctx context.Context, userID int64) ([]MasterListItem, error)
	ChangeStatus(ctx context.Context, appointmentID, masterUserID int64, newStatus string) (string, int64, error)
	GetBusyIntervals(ctx context.Context, masterID int64, dateStart, dateEnd time.Time) ([]BusyInterval, error)
	GetWorkingInterval(ctx context.Context, masterID int64, dayOfWeek int) (time.Time, time.Time, error)
	GetServiceDuration(ctx context.Context, serviceID, masterID int64) (int, error)
}
