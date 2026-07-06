package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yourorg/panel/internal/auth"
)

type ScheduleHandler struct {
	DB       *pgxpool.Pool
	Subusers *auth.SubuserChecker
}

type scheduleTask struct {
	Action            string `json:"action"`
	Payload           string `json:"payload"`
	TimeOffsetSeconds int    `json:"time_offset_seconds"`
}

type scheduleSummary struct {
	ID             int64          `json:"id"`
	Name           string         `json:"name"`
	CronMinute     string         `json:"cron_minute"`
	CronHour       string         `json:"cron_hour"`
	CronDayOfWeek  string         `json:"cron_day_of_week"`
	CronDayOfMonth string         `json:"cron_day_of_month"`
	IsActive       bool           `json:"is_active"`
	OnlyWhenOnline bool           `json:"only_when_online"`
	LastRunAt      *time.Time     `json:"last_run_at"`
	Tasks          []scheduleTask `json:"tasks"`
}

func (h *ScheduleHandler) resolveServer(w http.ResponseWriter, r *http.Request, permission string) (int64, bool) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return 0, false
	}

	var serverID, ownerID int64
	if err := h.DB.QueryRow(r.Context(),
		`SELECT id, owner_id FROM servers WHERE uuid = $1`, chi.URLParam(r, "uuid"),
	).Scan(&serverID, &ownerID); err != nil {
		http.Error(w, "server not found", http.StatusNotFound)
		return 0, false
	}
	if !h.Subusers.CanAccessServer(r.Context(), claims, ownerID, serverID, permission) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return 0, false
	}

	return serverID, true
}

func (h *ScheduleHandler) List(w http.ResponseWriter, r *http.Request) {
	serverID, ok := h.resolveServer(w, r, auth.PermSchedulesRead)
	if !ok {
		return
	}

	rows, err := h.DB.Query(r.Context(), `
		SELECT id, name, cron_minute, cron_hour, cron_day_of_week, cron_day_of_month,
		       is_active, only_when_online, last_run_at
		FROM server_schedules WHERE server_id = $1 ORDER BY created_at DESC`, serverID)
	if err != nil {
		http.Error(w, "failed to list schedules", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	schedules := make([]scheduleSummary, 0)
	for rows.Next() {
		var s scheduleSummary
		if err := rows.Scan(&s.ID, &s.Name, &s.CronMinute, &s.CronHour, &s.CronDayOfWeek,
			&s.CronDayOfMonth, &s.IsActive, &s.OnlyWhenOnline, &s.LastRunAt); err != nil {
			http.Error(w, "failed to read schedules", http.StatusInternalServerError)
			return
		}
		schedules = append(schedules, s)
	}
	rows.Close()

	for i := range schedules {
		taskRows, err := h.DB.Query(r.Context(),
			`SELECT action, payload, time_offset_seconds FROM schedule_tasks
			 WHERE schedule_id = $1 ORDER BY sequence_id`, schedules[i].ID)
		if err != nil {
			continue
		}
		tasks := make([]scheduleTask, 0)
		for taskRows.Next() {
			var t scheduleTask
			if err := taskRows.Scan(&t.Action, &t.Payload, &t.TimeOffsetSeconds); err == nil {
				tasks = append(tasks, t)
			}
		}
		taskRows.Close()
		schedules[i].Tasks = tasks
	}

	writeJSON(w, http.StatusOK, schedules)
}

type createScheduleRequest struct {
	Name           string         `json:"name"`
	CronMinute     string         `json:"cron_minute"`
	CronHour       string         `json:"cron_hour"`
	CronDayOfWeek  string         `json:"cron_day_of_week"`
	CronDayOfMonth string         `json:"cron_day_of_month"`
	OnlyWhenOnline bool           `json:"only_when_online"`
	Tasks          []scheduleTask `json:"tasks"`
}

func (h *ScheduleHandler) Create(w http.ResponseWriter, r *http.Request) {
	serverID, ok := h.resolveServer(w, r, auth.PermSchedulesWrite)
	if !ok {
		return
	}

	var req createScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || len(req.Tasks) == 0 {
		http.Error(w, "name and at least one task are required", http.StatusBadRequest)
		return
	}
	for _, f := range []*string{&req.CronMinute, &req.CronHour, &req.CronDayOfWeek, &req.CronDayOfMonth} {
		if *f == "" {
			*f = "*"
		}
	}

	ctx := r.Context()
	tx, err := h.DB.Begin(ctx)
	if err != nil {
		http.Error(w, "failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	var scheduleID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO server_schedules (server_id, name, cron_minute, cron_hour, cron_day_of_week,
		                               cron_day_of_month, only_when_online)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		serverID, req.Name, req.CronMinute, req.CronHour, req.CronDayOfWeek, req.CronDayOfMonth, req.OnlyWhenOnline,
	).Scan(&scheduleID)
	if err != nil {
		http.Error(w, "failed to create schedule", http.StatusInternalServerError)
		return
	}

	for i, t := range req.Tasks {
		if _, err := tx.Exec(ctx, `
			INSERT INTO schedule_tasks (schedule_id, sequence_id, action, payload, time_offset_seconds)
			VALUES ($1, $2, $3, $4, $5)`,
			scheduleID, i, t.Action, t.Payload, t.TimeOffsetSeconds,
		); err != nil {
			http.Error(w, "failed to create schedule task", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		http.Error(w, "failed to commit schedule", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"id": scheduleID})
}

func (h *ScheduleHandler) Toggle(w http.ResponseWriter, r *http.Request) {
	serverID, ok := h.resolveServer(w, r, auth.PermSchedulesWrite)
	if !ok {
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if _, err := h.DB.Exec(r.Context(),
		`UPDATE server_schedules SET is_active = NOT is_active WHERE id = $1 AND server_id = $2`,
		id, serverID,
	); err != nil {
		http.Error(w, "failed to toggle schedule", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ScheduleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	serverID, ok := h.resolveServer(w, r, auth.PermSchedulesWrite)
	if !ok {
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if _, err := h.DB.Exec(r.Context(),
		`DELETE FROM server_schedules WHERE id = $1 AND server_id = $2`, id, serverID,
	); err != nil {
		http.Error(w, "failed to delete schedule", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
