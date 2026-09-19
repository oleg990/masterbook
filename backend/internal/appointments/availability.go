package appointments

import (
	"net/http"
	"strconv"
	"time"
)

type AvailabilitySlot struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

func (h *Handler) GetAvailability(w http.ResponseWriter, r *http.Request) {
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
			"error": "invalid date, expected YYYY-MM-DD",
		})
		return
	}

	// Go: Sunday = 0, Monday = 1 ... Saturday = 6.
	// В нашей БД: Monday = 1 ... Sunday = 7.
	dayOfWeek := int(date.Weekday())
	if dayOfWeek == 0 {
		dayOfWeek = 7
	}

	var startTime time.Time
	var endTime time.Time

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
	).Scan(&startTime, &endTime)

	if err != nil {
		writeJSON(w, http.StatusOK, []AvailabilitySlot{})
		return
	}

	// Получаем существующие записи мастера на выбранную дату.
	rows, err := h.DB.Query(
		r.Context(),
		`
		SELECT start_time, end_time
		FROM appointments
		WHERE master_id = $1
		  AND status IN ('pending', 'confirmed')
		  AND start_time < $2
		  AND end_time > $3
		`,
		masterID,
		date.Add(24*time.Hour),
		date,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get appointments",
		})
		return
	}

	defer rows.Close()

	type busyInterval struct {
		start time.Time
		end   time.Time
	}

	var busy []busyInterval

	for rows.Next() {
		var item busyInterval

		if err := rows.Scan(&item.start, &item.end); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "failed to read appointments",
			})
			return
		}

		busy = append(busy, item)
	}

	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to read appointments",
		})
		return
	}

	// Временные интервалы в working_hours.
	startMinutes := startTime.Hour()*60 + startTime.Minute()
	endMinutes := endTime.Hour()*60 + endTime.Minute()

	result := make([]AvailabilitySlot, 0)

	// Пока используем шаг 30 минут.
	for minutes := startMinutes; minutes+30 <= endMinutes; minutes += 30 {
		slotStart := time.Date(
			date.Year(),
			date.Month(),
			date.Day(),
			minutes/60,
			minutes%60,
			0,
			0,
			date.Location(),
		)

		slotEnd := slotStart.Add(30 * time.Minute)

		isBusy := false

		for _, appointment := range busy {
			if slotStart.Before(appointment.end) &&
				slotEnd.After(appointment.start) {
				isBusy = true
				break
			}
		}

		if isBusy {
			continue
		}

		result = append(result, AvailabilitySlot{
			StartTime: slotStart.Format(time.RFC3339),
			EndTime:   slotEnd.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, result)
}