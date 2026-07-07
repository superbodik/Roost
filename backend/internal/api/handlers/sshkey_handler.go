package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/ssh"

	"github.com/yourorg/panel/internal/auth"
)

type SSHKeyHandler struct {
	DB *pgxpool.Pool
}

type sshKeySummary struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Fingerprint string    `json:"fingerprint"`
	CreatedAt   time.Time `json:"created_at"`
}

func (h *SSHKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rows, err := h.DB.Query(r.Context(),
		`SELECT id, name, fingerprint, created_at FROM ssh_keys WHERE user_id = $1 ORDER BY created_at DESC`,
		claims.UserID)
	if err != nil {
		http.Error(w, "failed to list ssh keys", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	keys := make([]sshKeySummary, 0)
	for rows.Next() {
		var k sshKeySummary
		if err := rows.Scan(&k.ID, &k.Name, &k.Fingerprint, &k.CreatedAt); err != nil {
			http.Error(w, "failed to read ssh keys", http.StatusInternalServerError)
			return
		}
		keys = append(keys, k)
	}

	writeJSON(w, http.StatusOK, keys)
}

type createSSHKeyRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

func (h *SSHKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req createSSHKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.PublicKey == "" {
		http.Error(w, "name and public_key are required", http.StatusBadRequest)
		return
	}

	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(strings.TrimSpace(req.PublicKey)))
	if err != nil {
		http.Error(w, "not a valid SSH public key", http.StatusBadRequest)
		return
	}
	fingerprint := ssh.FingerprintSHA256(pubKey)

	var id int64
	var createdAt time.Time
	if err := h.DB.QueryRow(r.Context(), `
		INSERT INTO ssh_keys (user_id, name, public_key, fingerprint)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`,
		claims.UserID, req.Name, strings.TrimSpace(req.PublicKey), fingerprint,
	).Scan(&id, &createdAt); err != nil {
		http.Error(w, "failed to save ssh key", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, sshKeySummary{ID: id, Name: req.Name, Fingerprint: fingerprint, CreatedAt: createdAt})
}

func (h *SSHKeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if _, err := h.DB.Exec(r.Context(),
		`DELETE FROM ssh_keys WHERE id = $1 AND user_id = $2`, id, claims.UserID,
	); err != nil {
		http.Error(w, "failed to delete ssh key", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
