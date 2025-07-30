package routes

import (
	"backend/internal/config"
	"backend/internal/service"
	"log/slog"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

type Handler struct {
	services *service.Service
}

func NewHandler(services *service.Service) *Handler {
	return &Handler{
		services: services,
	}
}

func (h *Handler) RegisterRoutes(router *chi.Mux, log *slog.Logger, cfg *config.Config) {
	router.Get("/swagger/*", httpSwagger.WrapHandler)

	router.Route("/api", func(r chi.Router) {
		r.Route("/tasks", func(r chi.Router) {
			r.Post("/create", h.createTask(log))
			r.Post("/{id}/add-link", h.addLink(log, cfg))
			r.Get("/{id}/status", h.getStatuses(log, cfg))
		})

		r.Route("/archives", func(r chi.Router) {
			r.Get("/{id}/download", h.downloadArchive(log))
		})
	})
}
