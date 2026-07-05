package handlers

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ActivityHandler struct {
	DB *pgxpool.Pool
}

type activityEntry struct {
	ID        int64     `json:"id"`
	Username  *string   `json:"username"`
	Event     string    `json:"event"`
	IPAddress *string   `json:"ip_address"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *ActivityHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `
		SELECT a.id, u.username, a.event, HOST(a.ip_address), a.created_at
		FROM activity_logs a
		LEFT JOIN users u ON u.id = a.actor_user_id
		ORDER BY a.created_at DESC
		LIMIT 100`)
	if err != nil {
		http.Error(w, "failed to list activity", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	entries := make([]activityEntry, 0)
	for rows.Next() {
		var e activityEntry
		if err := rows.Scan(&e.ID, &e.Username, &e.Event, &e.IPAddress, &e.CreatedAt); err != nil {
			http.Error(w, "failed to read activity", http.StatusInternalServerError)
			return
		}
		entries = append(entries, e)
	}

	writeJSON(w, http.StatusOK, entries)
}
