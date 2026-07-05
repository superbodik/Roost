package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AllocationHandler struct {
	DB *pgxpool.Pool
}

type allocationSummary struct {
	ID       int64   `json:"id"`
	NodeID   int64   `json:"node_id"`
	IP       string  `json:"ip"`
	Port     int     `json:"port"`
	Alias    *string `json:"alias"`
	ServerID *int64  `json:"server_id"`
}

func (h *AllocationHandler) List(w http.ResponseWriter, r *http.Request) {
	nodeIDStr := r.URL.Query().Get("node_id")
	nodeID, err := strconv.ParseInt(nodeIDStr, 10, 64)
	if err != nil {
		http.Error(w, "node_id query param is required", http.StatusBadRequest)
		return
	}
	freeOnly := r.URL.Query().Get("free") == "true"

	query := `SELECT id, node_id, HOST(ip), port, alias, server_id FROM allocations WHERE node_id = $1`
	if freeOnly {
		query += ` AND server_id IS NULL`
	}
	query += ` ORDER BY port`

	rows, err := h.DB.Query(r.Context(), query, nodeID)
	if err != nil {
		http.Error(w, "failed to list allocations", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	allocations := make([]allocationSummary, 0)
	for rows.Next() {
		var a allocationSummary
		if err := rows.Scan(&a.ID, &a.NodeID, &a.IP, &a.Port, &a.Alias, &a.ServerID); err != nil {
			http.Error(w, "failed to read allocations", http.StatusInternalServerError)
			return
		}
		allocations = append(allocations, a)
	}

	writeJSON(w, http.StatusOK, allocations)
}

type createAllocationRequest struct {
	NodeID int64  `json:"node_id"`
	IP     string `json:"ip"`
	Port   int    `json:"port"`
	Alias  string `json:"alias"`
}

func (h *AllocationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createAllocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.NodeID == 0 || req.IP == "" || req.Port == 0 {
		http.Error(w, "node_id, ip and port are required", http.StatusBadRequest)
		return
	}

	var alias *string
	if req.Alias != "" {
		alias = &req.Alias
	}

	var id int64
	err := h.DB.QueryRow(r.Context(), `
		INSERT INTO allocations (node_id, ip, port, alias)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		req.NodeID, req.IP, req.Port, alias,
	).Scan(&id)
	if err != nil {
		http.Error(w, "failed to create allocation (duplicate ip:port on this node?)", http.StatusConflict)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"id": id})
}
