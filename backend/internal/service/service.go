package service

import (
	"backend/internal/config"
	"backend/internal/repository"
	"log/slog"
)

type Tasks interface {
	CreateTask() (int64, error)
	AppendLink(id int64, link string, log *slog.Logger, cfg *config.Config) error
	GetArchivePath(id int64) (string, error)
	GetTask(id int64) (*repository.Task, error)
	MakeArchive(task repository.Task) (repository.Task, error)
}

type Service struct {
	Tasks Tasks
}

func NewService(repositories *repository.Repositories) *Service {
	return &Service{
		Tasks: NewTasksService(repositories.Tasks),
	}
}
