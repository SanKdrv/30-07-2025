package routes

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// downloadArchive скачивает архив по ID задачи
//
// @Summary      Скачать архив по ID задачи
// @Description  Скачивает архив, связанный с задачей, по её ID
// @Tags         archives
// @Param        id   path      int64  true  "Task ID"
// @Produce      application/zip
// @Success      200  {file}  archive.zip
// @Failure      400  {string}  string  "Неверный ID задачи"
// @Failure      404  {string}  string  "Архив не найден"
// @Failure      500  {string}  string  "Ошибка открытия файла"
// @Router       /api/archives/{id}/download [get]
func (h *Handler) downloadArchive(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Error("Неверный ID задачи", slog.String("error", err.Error()))
			http.Error(w, "Неверный ID задачи", http.StatusBadRequest)
			return
		}

		path, err := h.services.Tasks.GetArchivePath(id)
		if err != nil {
			log.Error("Ошибка получения пути к архиву", slog.String("error", err.Error()))
			http.Error(w, "Архив не найден", http.StatusNotFound)
			return
		}

		file, err := http.Dir("./").Open(path)
		if err != nil {
			log.Error("Ошибка открытия файла архива", slog.String("error", err.Error()))
			http.Error(w, "Ошибка открытия файла", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		w.Header().Set("Content-Disposition", "attachment; filename=\"archive.zip\"")
		w.Header().Set("Content-Type", "application/zip")
		http.ServeContent(w, r, "archive.zip", time.Time{}, file)
	}
}
