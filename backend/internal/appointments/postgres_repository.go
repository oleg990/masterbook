package appointments

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) Repository { return &postgresRepository{db: db} }

func (r *postgresRepository) Create(ctx context.Context, clientID, masterID, serviceID int64, startTime, endTime time.Time) (Appointment, int64, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Appointment{}, 0, err
	}
	defer tx.Rollback(ctx)
	var masterUserID int64
	if err := tx.QueryRow(ctx, `SELECT user_id FROM master_profiles WHERE id=$1 FOR UPDATE`, masterID).Scan(&masterUserID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Appointment{}, 0, ErrNotFound
		}
		return Appointment{}, 0, err
	}
	var duration int
	if err := tx.QueryRow(ctx, `SELECT duration_minutes FROM services WHERE id=$1 AND master_id=$2`, serviceID, masterID).Scan(&duration); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Appointment{}, 0, ErrServiceNotFound
		}
		return Appointment{}, 0, err
	}
	if !endTime.Equal(startTime.Add(time.Duration(duration) * time.Minute)) {
		return Appointment{}, 0, ErrConflict
	}
	var conflict bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM appointments WHERE master_id=$1 AND status IN ('pending','confirmed') AND start_time < $3 AND end_time > $2)`, masterID, startTime, endTime).Scan(&conflict); err != nil {
		return Appointment{}, 0, err
	}
	if conflict {
		return Appointment{}, 0, ErrConflict
	}
	var a Appointment
	err = tx.QueryRow(ctx, `INSERT INTO appointments(client_id,master_id,service_id,start_time,end_time,status)VALUES($1,$2,$3,$4,$5,'pending')RETURNING id,master_id,service_id,start_time,end_time,status`, clientID, masterID, serviceID, startTime, endTime).Scan(&a.ID, &a.MasterID, &a.ServiceID, &a.StartTime, &a.EndTime, &a.Status)
	if err != nil {
		return Appointment{}, 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Appointment{}, 0, err
	}
	return a, masterUserID, nil
}

func (r *postgresRepository) ListByClient(ctx context.Context, clientID int64) ([]ListItem, error) {
	rows, err := r.db.Query(ctx, `SELECT a.id,a.master_id,a.service_id,u.name,s.name,a.start_time,a.end_time,a.status FROM appointments a JOIN master_profiles mp ON mp.id=a.master_id JOIN users u ON u.id=mp.user_id JOIN services s ON s.id=a.service_id WHERE a.client_id=$1 ORDER BY a.start_time DESC`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ListItem, 0)
	for rows.Next() {
		var x ListItem
		if err := rows.Scan(&x.ID, &x.MasterID, &x.ServiceID, &x.MasterName, &x.ServiceName, &x.StartTime, &x.EndTime, &x.Status); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *postgresRepository) CancelByClient(ctx context.Context, appointmentID, clientID int64) (string, error) {
	var status string
	err := r.db.QueryRow(ctx, `UPDATE appointments SET status='cancelled',updated_at=NOW() WHERE id=$1 AND client_id=$2 AND status IN ('pending','confirmed') RETURNING status`, appointmentID, clientID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return status, err
}
func (r *postgresRepository) ListByMasterUser(ctx context.Context, userID int64) ([]MasterListItem, error) {
	rows, err := r.db.Query(ctx, `SELECT a.id,u.name,u.email,s.name,a.start_time,a.end_time,a.status FROM appointments a JOIN master_profiles mp ON mp.id=a.master_id JOIN users u ON u.id=a.client_id JOIN services s ON s.id=a.service_id WHERE mp.user_id=$1 ORDER BY a.start_time DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]MasterListItem, 0)
	for rows.Next() {
		var x MasterListItem
		if err := rows.Scan(&x.ID, &x.ClientName, &x.ClientEmail, &x.ServiceName, &x.StartTime, &x.EndTime, &x.Status); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (r *postgresRepository) ChangeStatus(ctx context.Context, appointmentID, masterUserID int64, newStatus string) (string, int64, error) {
	var condition string
	switch newStatus {
	case "confirmed":
		condition = "a.status='pending'"
	case "completed":
		condition = "a.status='confirmed'"
	case "cancelled":
		condition = "a.status IN ('pending','confirmed')"
	default:
		return "", 0, ErrInvalidStatus
	}
	q := `UPDATE appointments a SET status=$1,updated_at=NOW() FROM master_profiles mp WHERE a.id=$2 AND a.master_id=mp.id AND mp.user_id=$3 AND ` + condition + ` RETURNING a.status,a.client_id`
	var status string
	var clientID int64
	err := r.db.QueryRow(ctx, q, newStatus, appointmentID, masterUserID).Scan(&status, &clientID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", 0, ErrInvalidStatus
	}
	return status, clientID, err
}
func (r *postgresRepository) GetBusyIntervals(ctx context.Context, masterID int64, dateStart, dateEnd time.Time) ([]BusyInterval, error) {
	rows, err := r.db.Query(ctx, `SELECT start_time,end_time FROM appointments WHERE master_id=$1 AND status IN ('pending','confirmed') AND start_time<$2 AND end_time>$3`, masterID, dateEnd, dateStart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]BusyInterval, 0)
	for rows.Next() {
		var b BusyInterval
		if err := rows.Scan(&b.Start, &b.End); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
func (r *postgresRepository) GetWorkingInterval(ctx context.Context, masterID int64, dayOfWeek int) (time.Time, time.Time, error) {
	var start, end time.Time
	err := r.db.QueryRow(ctx, `SELECT start_time,end_time FROM working_hours WHERE master_id=$1 AND day_of_week=$2`, masterID, dayOfWeek).Scan(&start, &end)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, time.Time{}, ErrNotFound
	}
	return start, end, err
}
func (r *postgresRepository) GetServiceDuration(ctx context.Context, serviceID, masterID int64) (int, error) {
	var d int
	err := r.db.QueryRow(ctx, `SELECT duration_minutes FROM services WHERE id=$1 AND master_id=$2`, serviceID, masterID).Scan(&d)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrServiceNotFound
	}
	return d, err
}
