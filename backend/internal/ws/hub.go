package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/yourorg/panel/internal/models"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Hub struct {
	mu      sync.RWMutex
	rooms   map[uuid.UUID]map[*websocket.Conn]struct{}
	pollers map[uuid.UUID]context.CancelFunc

	FetchStats func(ctx context.Context, serverUUID uuid.UUID) (*models.ResourceStats, error)
}

func NewHub() *Hub {
	return &Hub{
		rooms:   make(map[uuid.UUID]map[*websocket.Conn]struct{}),
		pollers: make(map[uuid.UUID]context.CancelFunc),
	}
}

func (h *Hub) ServeServerSocket(w http.ResponseWriter, r *http.Request, serverUUID uuid.UUID) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	h.subscribe(serverUUID, conn)
	defer h.unsubscribe(serverUUID, conn)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *Hub) Broadcast(serverUUID uuid.UUID, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ws broadcast marshal failed: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	for conn := range h.rooms[serverUUID] {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("ws write failed: %v", err)
		}
	}
}

func (h *Hub) subscribe(serverUUID uuid.UUID, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[serverUUID] == nil {
		h.rooms[serverUUID] = make(map[*websocket.Conn]struct{})
	}
	firstSubscriber := len(h.rooms[serverUUID]) == 0
	h.rooms[serverUUID][conn] = struct{}{}

	if firstSubscriber && h.FetchStats != nil {
		ctx, cancel := context.WithCancel(context.Background())
		h.pollers[serverUUID] = cancel
		go h.pollStats(ctx, serverUUID)
	}
}

func (h *Hub) unsubscribe(serverUUID uuid.UUID, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms[serverUUID], conn)
	if len(h.rooms[serverUUID]) == 0 {
		delete(h.rooms, serverUUID)
		if cancel, ok := h.pollers[serverUUID]; ok {
			cancel()
			delete(h.pollers, serverUUID)
		}
	}
}

func (h *Hub) pollStats(ctx context.Context, serverUUID uuid.UUID) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats, err := h.FetchStats(ctx, serverUUID)
			if err != nil {
				continue
			}
			h.Broadcast(serverUUID, stats)
		}
	}
}
