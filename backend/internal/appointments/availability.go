package appointments

import (
	"net/http"
	"strconv"
	"time"
)

func (h *Handler) Availability(w http.ResponseWriter, r *http.Request) {
	masterID, err := strconv.ParseInt(
		r.PathValue("masterID"),
		10,
		64,
	)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid master id",
		})
		return
	}

	dateString := r.URL.Query().Get("date")

	if dateString == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "date is required",
		})
		return
	}

	date, err := time.Parse("2006-01-02", dateString)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "date must have format YYYY-MM-DD",
		})
		return
	}

	dayOfWeek := int(date.Weekday())

	if dayOfWeek == 0 {
		dayOfWeek = 7
	}

	var (
		startTime time.Time
		endTime   time.Time
	)

	err = h.DB.QueryRow(
		r.Context(),
		`
		SELECT start_time, end_time
		FROM working_hours
		WHERE master_id = $1
		  AND day_of_week = $2
		`,
		masterID,
		dayOfWeek,
	).Scan(
		&startTime,
		&endTime,
	)

	if err != nil {
		writeJSON(w, http.StatusOK, []string{})
		return
	}

	duration := 30 * time.Minute

	slots := make([]string, 0)

	current := time.Date(
		date.Year(),
		date.Month(),
		date.Day(),
		startTime.Hour(),
		startTime.Minute(),
		0,
		0,
		date.Location(),
	)

	end := time.Date(
		date.Year(),
		date.Month(),
		date.Day(),
		endTime.Hour(),
		endTime.Minute(),
		0,
		0,
		date.Location(),
	)

	appointments, err := h.getAppointmentsForDay(
		r,
		masterID,
		date,
	)

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get appointments",
		})
		return
	}

	for current.Before(end) {
		slotEnd := current.Add(duration)

		if slotEnd.After(end) {
			break
		}

		if !hasConflict(current, slotEnd, appointments) {
			slots = append(
				slots,
				current.Format("15:04"),
			)
		}

		current = current.Add(duration)
	}

	writeJSON(w, http.StatusOK, slots)
}

type appointmentTime struct {
	Start time.Time
	End   time.Time
}

func (h *Handler) getAppointmentsForDay(
	r *http.Request,
	masterID int64,
	date time.Time,
) ([]appointmentTime, error) {
	startOfDay := time.Date(
		date.Year(),
		date.Month(),
		date.Day(),
		0,
		0,
		0,
		0,
		date.Location(),
	)

	endOfDay := startOfDay.Add(24 * time.Hour)

	rows, err := h.DB.Query(
		r.Context(),
		`
		SELECT start_time, end_time
		FROM appointments
		WHERE master_id = $1
		  AND status IN ('pending', 'confirmed')
		  AND start_time < $3
		  AND end_time > $2
		ORDER BY start_time
		`,
		masterID,
		startOfDay,
		endOfDay,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	result := make([]appointmentTime, 0)

	for rows.Next() {
		var item appointmentTime

		if err := rows.Scan(
			&item.Start,
			&item.End,
		); err != nil {
			return nil, err
		}

		result = append(result, item)
	}

	return result, nil
}

func hasConflict(
	start time.Time,
	end time.Time,
	appointments []appointmentTime,
) bool {
	for _, appointment := range appointments {
		if start.Before(appointment.End) &&
			end.After(appointment.Start) {
			return true
		}
	}

	return false
}
