package repository

import (
	"fmt"
	"sync"
)

const (
	TaskCreated    string = "Создано"
	TaskProcessing string = "Обрабатывается"
	TaskCompleted  string = "Выполнено"
	TaskFailed     string = "Ошибка"
)

type Task struct {
	Id               int64    `json:"-"`
	Status           string   `json:"status"`
	Links            []string `json:"-"`
	LoadedFilesLinks []string `json:"-"`
	ArchivePath      string   `json:"-"`
	Errors           []string `json:"errors,omitempty"`
}

type TasksRepository struct {
	// semaphore chan struct{}
	tasks map[int64]Task
	mu    sync.Mutex
	ptr   int64
}

func NewTasksRepository() *TasksRepository {
	return &TasksRepository{tasks: make(map[int64]Task)}
	// return &TasksRepository{semaphore: make(chan struct{}, 3), tasks: make(map[int64]Task)}
}

func (r *TasksRepository) CreateTask() (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.ptr
	r.ptr++

	_, exists := r.tasks[id]
	if exists {
		return -1, fmt.Errorf("задача с идентификатором %d уже создана", id)
	}

	r.tasks[id] = Task{
		Id:     id,
		Status: TaskCreated,
	}
	return id, nil
}

func (r *TasksRepository) AppendLink(id int64, link string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, exists := r.tasks[id]
	if !exists {
		return fmt.Errorf("задача с идентификатором %d не найдена", id)
	}

	task.Links = append(task.Links, link)
	r.tasks[id] = task
	return nil
}

func (r *TasksRepository) AppendLoadedFileLink(id int64, link string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, exists := r.tasks[id]
	if !exists {
		return fmt.Errorf("задача с идентификатором %d не найдена", id)
	}

	task.LoadedFilesLinks = append(task.LoadedFilesLinks, link)
	r.tasks[id] = task
	return nil
}

func (r *TasksRepository) AppendError(id int64, err string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	task, exists := r.tasks[id]
	if !exists {
		return fmt.Errorf("задача с идентификатором %d не найдена", id)
	}
	task.Errors = append(task.Errors, err)
	r.tasks[id] = task
	return nil
}

func (r *TasksRepository) UpdateTaskStatus(id int64, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, exists := r.tasks[id]
	if !exists {
		return fmt.Errorf("задача с идентификатором %d не найдена", id)
	}

	task.Status = status
	r.tasks[id] = task
	return nil
}

func (r *TasksRepository) GetTask(id int64) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, exists := r.tasks[id]
	if !exists {
		return Task{}, fmt.Errorf("задача с идентификатором %d не найдена", id)
	}

	return task, nil
}

func (r *TasksRepository) CountActiveTasks() int8 {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	for _, task := range r.tasks {
		if task.Status == TaskCreated || task.Status == TaskProcessing {
			count++
		}
		if count > 3 {
			return 3
		}
	}
	return int8(count)
}

func (r *TasksRepository) UpdateArchiveName(id int64, archiveName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, exists := r.tasks[id]
	if !exists {
		return fmt.Errorf("задача с идентификатором %d не найдена", id)
	}

	task.ArchivePath = archiveName
	r.tasks[id] = task
	return nil
}
