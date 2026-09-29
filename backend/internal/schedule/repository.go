package schedule

import (
	"context"
	"errors"
	"time"
)

type WorkingHour struct {
	ID        int64
	MasterID  int64
	DayOfWeek int
	StartTime time.Time
	EndTime   time.Time
}

var ErrNotFound = errors.New("working hour not found")

type Repository interface {
	Upsert(ctx context.Context, masterID int64, day int, startTime, endTime string) (WorkingHour, error)
	ListByMaster(ctx context.Context, masterID int64) ([]WorkingHour, error)
	FindByMasterAndDay(ctx context.Context, masterID int64, day int) (WorkingHour, error)
	DeleteByMasterAndDay(ctx context.Context, masterID int64, day int) error
}
