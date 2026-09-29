package appointments

import (
	"context"
	"time"
)

type NotificationSender interface {
	Create(ctx context.Context, userID int64, title, message, notificationType string) error
}

type Service struct {
	repo          Repository
	notifications NotificationSender
}

func NewService(repo Repository, notifications NotificationSender) *Service {
	return &Service{repo: repo, notifications: notifications}
}

func (s *Service) Create(ctx context.Context, clientID, masterID, serviceID int64, start time.Time) (Appointment, error) {
	if masterID <= 0 || serviceID <= 0 {
		return Appointment{}, ErrNotFound
	}
	duration, err := s.repo.GetServiceDuration(ctx, serviceID, masterID)
	if err != nil {
		return Appointment{}, err
	}
	end := start.Add(time.Duration(duration) * time.Minute)
	day := int(start.Weekday())
	if day == 0 {
		day = 7
	}
	workStart, workEnd, err := s.repo.GetWorkingInterval(ctx, masterID, day)
	if err != nil {
		return Appointment{}, ErrOutsideWorkingHours
	}
	// PostgreSQL TIME is returned with an arbitrary date; compare minutes from midnight.
	startMinutes := start.Hour()*60 + start.Minute()
	endMinutes := end.Hour()*60 + end.Minute()
	workStartMinutes := workStart.Hour()*60 + workStart.Minute()
	workEndMinutes := workEnd.Hour()*60 + workEnd.Minute()
	if startMinutes < workStartMinutes || endMinutes > workEndMinutes || !start.Before(end) {
		return Appointment{}, ErrOutsideWorkingHours
	}
	appointment, masterUserID, err := s.repo.Create(ctx, clientID, masterID, serviceID, start, end)
	if err != nil {
		return Appointment{}, err
	}
	if s.notifications != nil {
		_ = s.notifications.Create(ctx, masterUserID, "Новая запись", "К вам поступила новая запись клиента.", "appointment_created")
	}
	return appointment, nil
}
func (s *Service) ListMine(ctx context.Context, clientID int64) ([]ListItem, error) {
	return s.repo.ListByClient(ctx, clientID)
}
func (s *Service) Cancel(ctx context.Context, appointmentID, clientID int64) (string, error) {
	return s.repo.CancelByClient(ctx, appointmentID, clientID)
}
func (s *Service) ListMasterAppointments(ctx context.Context, userID int64) ([]MasterListItem, error) {
	return s.repo.ListByMasterUser(ctx, userID)
}
func (s *Service) ChangeStatus(ctx context.Context, appointmentID, masterUserID int64, newStatus string) (string, error) {
	status, clientID, err := s.repo.ChangeStatus(ctx, appointmentID, masterUserID, newStatus)
	if err != nil {
		return "", err
	}
	if s.notifications != nil {
		switch newStatus {
		case "confirmed":
			_ = s.notifications.Create(ctx, clientID, "Запись подтверждена", "Мастер подтвердил вашу запись.", "appointment_confirmed")
		case "cancelled":
			_ = s.notifications.Create(ctx, clientID, "Запись отменена", "Мастер отменил вашу запись.", "appointment_cancelled")
		}
	}
	return status, nil
}
func (s *Service) Availability(ctx context.Context, masterID, serviceID int64, date time.Time) ([]AvailabilitySlot, error) {
	duration := 30
	if serviceID > 0 {
		var err error
		duration, err = s.repo.GetServiceDuration(ctx, serviceID, masterID)
		if err != nil {
			return nil, err
		}
	}
	day := int(date.Weekday())
	if day == 0 {
		day = 7
	}
	workStart, workEnd, err := s.repo.GetWorkingInterval(ctx, masterID, day)
	if err != nil {
		return []AvailabilitySlot{}, nil
	}
	dateStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	dateEnd := dateStart.Add(24 * time.Hour)
	busy, err := s.repo.GetBusyIntervals(ctx, masterID, dateStart, dateEnd)
	if err != nil {
		return nil, err
	}
	startMinutes := workStart.Hour()*60 + workStart.Minute()
	endMinutes := workEnd.Hour()*60 + workEnd.Minute()
	result := make([]AvailabilitySlot, 0)
	// UI remains compatible with 30-minute start increments, but slot length follows the service.
	for minutes := startMinutes; minutes+duration <= endMinutes; minutes += 30 {
		slotStart := time.Date(date.Year(), date.Month(), date.Day(), minutes/60, minutes%60, 0, 0, date.Location())
		slotEnd := slotStart.Add(time.Duration(duration) * time.Minute)
		busySlot := false
		for _, b := range busy {
			if slotStart.Before(b.End) && slotEnd.After(b.Start) {
				busySlot = true
				break
			}
		}
		if !busySlot {
			result = append(result, AvailabilitySlot{StartTime: slotStart.Format(time.RFC3339), EndTime: slotEnd.Format(time.RFC3339)})
		}
	}
	return result, nil
}
