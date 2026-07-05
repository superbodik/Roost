package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/yourorg/panel-daemon/internal/console"
	"github.com/yourorg/panel-daemon/internal/docker"
)

func NewRouter(dockerManager *docker.Manager, consoleHub *console.Hub, daemonToken string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(RequireDaemonToken(daemonToken))

	h := &Handlers{Docker: dockerManager, Console: consoleHub}

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/servers", h.CreateServer)
		r.Post("/servers/{uuid}/power", h.Power)
		r.Delete("/servers/{uuid}", h.Delete)
		r.Get("/servers/{uuid}/stats", h.Stats)

		r.Get("/servers/{uuid}/files", h.ListFiles)
		r.Get("/servers/{uuid}/files/contents", h.ReadFile)
		r.Put("/servers/{uuid}/files/contents", h.WriteFile)
		r.Delete("/servers/{uuid}/files", h.DeleteFile)
		r.Post("/servers/{uuid}/files/directory", h.CreateDirectory)
		r.Post("/servers/{uuid}/files/rename", h.RenameFile)
	})
	r.Get("/ws/servers/{uuid}", h.ConsoleSocket)

	return r
}
