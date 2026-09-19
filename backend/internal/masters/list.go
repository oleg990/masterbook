package masters

import "net/http"

type MasterListItem struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	PhotoURL    *string `json:"photo_url"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(
		r.Context(),
		`
		SELECT
			mp.id,
			u.name,
			mp.description,
			mp.photo_url
		FROM master_profiles mp
		JOIN users u ON u.id = mp.user_id
		WHERE u.role = 'master'
		ORDER BY u.name
		`,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get masters",
		})
		return
	}

	defer rows.Close()

	result := make([]MasterListItem, 0)

	for rows.Next() {
		var item MasterListItem

		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.PhotoURL,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "failed to read masters",
			})
			return
		}

		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to read masters",
		})
		return
	}

	writeJSON(w, http.StatusOK, result)
}
