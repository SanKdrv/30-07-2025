package main

import (
	_ "backend/docs"
	"backend/internal/config"
	"backend/internal/lib/logger/slogpretty"
	"backend/internal/middleware/logger"
	"backend/internal/repository"
	"backend/internal/routes"
	"backend/internal/service"
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	// http.ListenAndServe()
	cfg := config.MustLoad()
	log := setupLogger(cfg.Environment)

	log.Info("Запуск приложения", slog.String("env", cfg.Environment))
	log.Debug("Отладочные сообщения включены")

	router := chi.NewRouter()

	// Настройка CORS middleware
	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // Разрешаем фронтенд на порту ...
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		Debug:            true,
	})

	repos := repository.NewRepositories()
	services := service.NewService(repos)
	handlers := routes.NewHandler(services)

	// Применяем CORS middleware
	router.Use(corsMiddleware.Handler)

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(logger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	handlers.RegisterRoutes(router, log, &cfg)

	log.Info("Запуск сервера", slog.String("address", cfg.HTTPServer.Address))

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.HTTPServer.Timeout),
		WriteTimeout: time.Duration(cfg.HTTPServer.Timeout),
		IdleTimeout:  time.Duration(cfg.HTTPServer.IdleTimeout),
	}

	if cfg.Environment == "development" {
		go func() {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Error("Не удалось запустить сервер", slog.String("error", err.Error()))
			}
		}()
	} else {
		return
	}
	log.Info("Сервер запущен")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Ошибка выключения: ", slog.String("error", err.Error()))
	}
	log.Info("Сервер остановлен")
	if err := os.RemoveAll("./backend/archives"); err != nil {
		log.Error("Ошибка удаления папки archives", slog.String("error", err.Error()))
	}
	if err := os.RemoveAll("./backend/static"); err != nil {
		log.Error("Ошибка удаления папки static", slog.String("error", err.Error()))
	}
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case "development":
		log = setupPrettySlog()
	case "prod":
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
