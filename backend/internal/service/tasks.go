package service

import (
	"archive/zip"
	"backend/internal/config"
	"backend/internal/repository"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type TasksService struct {
	semaphore chan struct{}
	repo      repository.Tasks
}

func NewTasksService(repo repository.Tasks) *TasksService {
	return &TasksService{
		semaphore: make(chan struct{}, 3),
		repo:      repo,
	}
}

func (s *TasksService) CreateTask() (int64, error) {
	if s.repo.CountActiveTasks() == 3 {
		return -1, fmt.Errorf("сервер в данный момент занят")
	}
	return s.repo.CreateTask()
}

func (s *TasksService) AppendLink(id int64, link string, log *slog.Logger, cfg *config.Config) error {
	if link == "" {
		return fmt.Errorf("ссылка не может быть пустой")
	}
	if !(strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://")) {
		return fmt.Errorf("ссылка должна начинаться с http:// или https://")
	}

	err := s.repo.AppendLink(id, link)
	if err != nil {
		return fmt.Errorf("не удалось добавить ссылку: %w", err)
	}

	if task, err := s.repo.GetTask(id); err != nil {
		return fmt.Errorf("не удалось получить задачу: %w", err)
	} else if task.Status != repository.TaskFailed {
		err = s.repo.UpdateTaskStatus(id, repository.TaskProcessing)
		if err != nil {
			return fmt.Errorf("не удалось обновить статус задачи: %w", err)
		}
	}

	go func(id int64, link string, s *TasksService, log *slog.Logger, cfg *config.Config) {
		s.semaphore <- struct{}{}
		s.DownloadFile(id, link, log, cfg)
	}(id, link, s, log, cfg)

	return nil
}

func (s *TasksService) DownloadFile(id int64, link string, log *slog.Logger, cfg *config.Config) {
	var err error
	defer func(s *TasksService) { <-s.semaphore }(s)
	loadedLink := ""

	pass := false
	for _, val := range strings.Split(cfg.AllowedExtensions, ",") {
		if strings.HasSuffix(link, val) {
			pass = true
			break
		}
	}

	if pass {
		resp, err := http.Get(link)
		if err != nil {
			// log.Error("Ошибка при скачивании файла", slog.String("error", err.Error()), slog.Int64("task_id", id))
			err = fmt.Errorf("ошибка при скачивании файла: %w", err)
			s.handleErr(err, id, log)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			// log.Error("Ошибка при скачивании файла", slog.String("error", fmt.Sprintf("статус: %d", resp.StatusCode)), slog.Int64("task_id", id))
			err = fmt.Errorf("файл не найден, статус: %d", resp.StatusCode)
			s.handleErr(err, id, log)
			return
		}

		fileName := fmt.Sprintf("%s/%d_%s", "./backend/static", id, getFileNameFromURL(link))

		if err := os.MkdirAll("./backend/static", os.ModePerm); err != nil {
			// log.Error("Ошибка при создании директории", slog.String("error", err.Error()), slog.Int64("task_id", id))
			err = fmt.Errorf("ошибка при создании директории: %w", err)
			s.handleErr(err, id, log)
			return
		}
		out, err := os.Create(fileName)
		if err != nil {
			// log.Error("Ошибка при создании файла", slog.String("error", err.Error()), slog.Int64("task_id", id))
			err = fmt.Errorf("ошибка при создании файла: %w", err)
			s.handleErr(err, id, log)
			return
		}
		defer out.Close()

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			// log.Error("Ошибка при сохранении файла", slog.String("error", err.Error()), slog.Int64("task_id", id))
			err = fmt.Errorf("ошибка при сохранении файла: %w", err)
			s.handleErr(err, id, log)
			return
		}

		loadedLink = fileName
	} else {
		// log.Error("Неверный формат файла", slog.String("link", link), slog.Int64("task_id", id))
		err = fmt.Errorf("неверный формат файла, поддерживаются только .pdf и .jpeg")
		s.handleErr(err, id, log)
		return
	}

	err = s.repo.AppendLoadedFileLink(id, loadedLink)
	if err != nil {
		// log.Error("Ошибка при добавлении ссылки на загруженный файл", slog.String("error", err.Error()), slog.Int64("task_id", id))
		err = fmt.Errorf("не удалось добавить ссылку на загруженный файл: %w", err)
		s.handleErr(err, id, log)
		return
	}
}

func (s *TasksService) handleErr(err error, id int64, log *slog.Logger) {
	if err != nil {
		info := fmt.Sprintf("ошибка при скачивании %d: %v\n", id, err)
		if updateErr := s.repo.AppendError(id, info); updateErr != nil {
			log.Error("не удалось записать ошибку задачи: ", strconv.FormatInt(id, 10), updateErr)
		}
		if updateErr := s.repo.UpdateTaskStatus(id, repository.TaskFailed); updateErr != nil {
			log.Error("не удалось обновить статус задачи: ", strconv.FormatInt(id, 10), updateErr)
		}
	}
}

func getFileNameFromURL(link string) string {
	parts := strings.Split(link, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func (s *TasksService) GetTask(id int64) (*repository.Task, error) {
	task, err := s.repo.GetTask(id)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить задачу: %w", err)
	}
	if len(task.LoadedFilesLinks)+len(task.Errors) >= 3 && len(task.LoadedFilesLinks) > 0 {
		err = s.repo.UpdateTaskStatus(id, repository.TaskCompleted)
		if err != nil {
			return nil, fmt.Errorf("не удалось обновить статус задачи: %w", err)
		}
	}
	return &task, nil
}

func (s *TasksService) MakeArchive(task repository.Task) (repository.Task, error) {
	archiveName := fmt.Sprintf("./backend/archives/%d_archive.zip", task.Id)
	if err := os.MkdirAll("./backend/archives", os.ModePerm); err != nil {
		err = fmt.Errorf("ошибка при создании директории: %w", err)
		return task, err
	}
	archiveFile, err := os.Create(archiveName)
	if err != nil {
		return task, fmt.Errorf("ошибка при создании архива: %w", err)
	}
	defer archiveFile.Close()
	zipWriter := zip.NewWriter(archiveFile)
	defer zipWriter.Close()

	for _, filePath := range task.LoadedFilesLinks {
		if err := addFileToZip(zipWriter, filePath); err != nil {
			return task, fmt.Errorf("ошибка при добавлении файла %s в архив: %w", filePath, err)
		}
	}

	if err := s.repo.UpdateArchiveName(task.Id, archiveName); err != nil {
		return task, fmt.Errorf("ошибка при обновлении имени архива: %w", err)
	}
	task.ArchivePath = archiveName

	return task, nil
}

func addFileToZip(zipWriter *zip.Writer, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("ошибка при открытии файла: %w", err)
	}
	defer file.Close()

	w, err := zipWriter.Create(getFileNameFromURL(filePath))
	if err != nil {
		return fmt.Errorf("ошибка при создании файла в архиве: %w", err)
	}

	_, err = io.Copy(w, file)
	if err != nil {
		return fmt.Errorf("ошибка при копировании файла в архив: %w", err)
	}

	return nil
}

func (s *TasksService) GetArchivePath(id int64) (string, error) {
	task, err := s.repo.GetTask(id)
	if err != nil {
		return "", fmt.Errorf("не удалось получить задачу: %w", err)
	}
	if task.ArchivePath == "" {
		return "", fmt.Errorf("архив для задачи %d не найден", id)
	}
	return task.ArchivePath, nil
}
