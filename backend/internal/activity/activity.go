package activity

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Entry struct {
	ActorUserID *int64
	ServerID    *int64
	NodeID      *int64
	Event       string
	IPAddress   string
	Metadata    map[string]any
}

func Record(ctx context.Context, pool *pgxpool.Pool, e Entry) {
	metadata := e.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		log.Printf("activity: marshal metadata failed: %v", err)
		return
	}

	var ip any
	if host := IPOnly(e.IPAddress); host != "" {
		ip = host
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO activity_logs (actor_user_id, server_id, node_id, event, ip_address, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		e.ActorUserID, e.ServerID, e.NodeID, e.Event, ip, metadataJSON)
	if err != nil {
		log.Printf("activity: insert failed: %v", err)
	}
}

func IPOnly(remoteAddr string) string {
	if remoteAddr == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}

func RequestIP(r *http.Request) string {
	return IPOnly(r.RemoteAddr)
}
