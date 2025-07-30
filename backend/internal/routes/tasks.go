package routes

import (
	"backend/internal/config"
	"backend/internal/repository"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Task *repository.Task

// createTask godoc
// @Summary      Создать новую задачу
// @Description  Создает новую задачу и возвращает её ID
// @Tags         tasks
// @Success      201 {string} string "Задача успешно создана с ID: {id}"
// @Failure      500 {string} string "Ошибка при создании задачи"
// @Router       /api/tasks/create [post]
func (h *Handler) createTask(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idx, err := h.services.Tasks.CreateTask()
		if err != nil {
			log.Error("Ошибка при создании задачи", slog.String("error", err.Error()))
			http.Error(w, "Ошибка при создании задачи", http.StatusInternalServerError)
			return
		}
		log.Info("Задача успешно создана", slog.Int64("task_id", idx))
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("Задача успешно создана с ID: " + strconv.FormatInt(idx, 10)))
	}
}

// addLink godoc
// @Summary      Добавить ссылку к задаче
// @Description  Добавляет ссылку к задаче по её ID
// @Tags         tasks
// @Param        id   path      int    true  "ID задачи"
// @Param        link formData  string true  "Ссылка для добавления"
// @Success      200  {string}  string "Ссылка успешно добавлена к задаче"
// @Failure      400  {string}  string "Неверный ID задачи или пустая ссылка"
// @Failure      500  {string}  string "Ошибка при добавлении ссылки к задаче"
// @Router       /api/tasks/{id}/add-link [post]
func (h *Handler) addLink(log *slog.Logger, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Error("Неверный ID задачи", slog.String("error", err.Error()))
			http.Error(w, "Неверный ID задачи", http.StatusBadRequest)
			return
		}

		link := r.FormValue("link")
		// log.Info("Получена ссылка для добавления", slog.Int64("task_id", id), slog.String("link", link))
		if link == "" {
			http.Error(w, "Ссылка не может быть пустой", http.StatusBadRequest)
			return
		}

		err = h.services.Tasks.AppendLink(id, link, log, cfg)
		if err != nil {
			log.Error("Ошибка при добавлении ссылки к задаче", slog.String("error", err.Error()))
			http.Error(w, "Ошибка при добавлении ссылки к задаче", http.StatusInternalServerError)
			return
		}

		log.Info("Ссылка успешно добавлена к задаче", slog.Int64("task_id", id), slog.String("link", link))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ссылка успешно добавлена к задаче"))
	}
}

type GetStatusesResponse struct {
	Task         Task   `json:"task"`
	DownloadLink string `json:"download_link,omitempty"`
}

// getStatuses godoc
// @Summary      Получить статусы задачи
// @Description  Возвращает статусы задачи по её ID. В случае, когда ни один файл не удалось скачать, архив не будет возвращён.
// @Description  Если задача завершена успешно/удалось установить хоть один файл на момент завершения, возвращает ссылку на скачивание архива
// @Tags         tasks
// @Param        id   path      int    true  "ID задачи"
// @Success      200  {object}  GetStatusesResponse   "Статусы задачи успешно получены"
// @Failure      400  {string}  string "Неверный ID задачи"
// @Failure      500  {string}  string "Ошибка при получении или сериализации статусов задачи"
// @Router       /api/tasks/{id}/status [get]
func (h *Handler) getStatuses(log *slog.Logger, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Error("Неверный ID задачи", slog.String("error", err.Error()))
			http.Error(w, "Неверный ID задачи", http.StatusBadRequest)
			return
		}

		task, err := h.services.Tasks.GetTask(id)
		if err != nil {
			log.Error("Ошибка при получении статусов задачи", slog.String("error", err.Error()))
			http.Error(w, "Ошибка при получении статусов задачи", http.StatusInternalServerError)
			return
		}

		log.Info("Статусы задачи успешно получены", slog.Int64("task_id", id))

		var link string

		if len(task.Links)+len(task.Errors) >= 3 && len(task.LoadedFilesLinks) > 0 && (task.Status == repository.TaskFailed || task.Status == repository.TaskCompleted) {
			link = fmt.Sprintf("http://localhost%s/api/archives/%d/download", cfg.HTTPServer.Address, id)
			_, err := h.services.Tasks.MakeArchive(*task)
			if err != nil {
				log.Error("Ошибка при создании архива", slog.String("error", err.Error()), slog.Int64("task_id", id))
				http.Error(w, "Ошибка при создании архива", http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		resp := GetStatusesResponse{
			Task:         task,
			DownloadLink: link,
		}
		bytes, err := json.Marshal(resp)
		if err != nil {
			log.Error("Ошибка при сериализации задачи", slog.String("error", err.Error()))
			http.Error(w, "Ошибка при сериализации задачи", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(bytes)
	}
}
