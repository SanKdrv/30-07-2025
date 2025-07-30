package repository

type Tasks interface {
	CreateTask() (int64, error)
	AppendLink(id int64, link string) error
	AppendLoadedFileLink(id int64, link string) error
	GetTask(id int64) (Task, error)
	UpdateTaskStatus(id int64, status string) error
	CountActiveTasks() int8
	AppendError(id int64, err string) error
	UpdateArchiveName(id int64, archiveName string) error
}

type Repositories struct {
	Tasks Tasks
}

func NewRepositories() *Repositories {
	return &Repositories{
		Tasks: NewTasksRepository(),
	}
}
