package handlers

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type EggHandler struct {
	DB *pgxpool.Pool
}

type eggSummary struct {
	ID             int    `json:"id"`
	Category       string `json:"category"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	DockerImage    string `json:"docker_image"`
	StartupCommand string `json:"startup_command"`
}

func (h *EggHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `
		SELECT id, category, name, description, docker_image, startup_command
		FROM eggs ORDER BY category, name`)
	if err != nil {
		http.Error(w, "failed to list eggs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	eggs := make([]eggSummary, 0)
	for rows.Next() {
		var e eggSummary
		if err := rows.Scan(&e.ID, &e.Category, &e.Name, &e.Description, &e.DockerImage, &e.StartupCommand); err != nil {
			http.Error(w, "failed to read eggs", http.StatusInternalServerError)
			return
		}
		eggs = append(eggs, e)
	}

	writeJSON(w, http.StatusOK, eggs)
}
