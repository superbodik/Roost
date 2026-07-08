package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
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
	var beforeID int64
	if raw := r.URL.Query().Get("before_id"); raw != "" {
		var err error
		beforeID, err = strconv.ParseInt(raw, 10, 64)
		if err != nil {
			http.Error(w, "invalid before_id", http.StatusBadRequest)
			return
		}
	}

	var rows pgx.Rows
	var err error
	if beforeID > 0 {
		rows, err = h.DB.Query(r.Context(), `
			SELECT a.id, u.username, a.event, HOST(a.ip_address), a.created_at
			FROM activity_logs a
			LEFT JOIN users u ON u.id = a.actor_user_id
			WHERE a.id < $1
			ORDER BY a.id DESC
			LIMIT 100`, beforeID)
	} else {
		rows, err = h.DB.Query(r.Context(), `
			SELECT a.id, u.username, a.event, HOST(a.ip_address), a.created_at
			FROM activity_logs a
			LEFT JOIN users u ON u.id = a.actor_user_id
			ORDER BY a.id DESC
			LIMIT 100`)
	}
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
