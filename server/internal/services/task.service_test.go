package services

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
)

// MockTaskRepository is a mock implementation of TaskRepositoryInterface.
type MockTaskRepository struct {
	createFunc         func(*models.Task) error
	getByIDFunc        func(string) (*models.Task, error)
	listFunc           func(repositories.TaskFilter) ([]models.Task, int64, error)
	updateFunc         func(*models.Task) error
	deleteFunc         func(string) error
	getChildrenFunc    func(string) ([]models.Task, error)
	getRootFunc        func(string) (*models.Task, error)
	getDescendantsFunc func(string) ([]models.Task, error)
}

func (m *MockTaskRepository) Create(task *models.Task) error {
	if m.createFunc != nil {
		return m.createFunc(task)
	}
	return nil
}

func (m *MockTaskRepository) GetByID(id string) (*models.Task, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(id)
	}
	return nil, errors.New("task not found")
}

func (m *MockTaskRepository) List(filter repositories.TaskFilter) ([]models.Task, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(filter)
	}
	return nil, 0, nil
}

func (m *MockTaskRepository) Update(task *models.Task) error {
	if m.updateFunc != nil {
		return m.updateFunc(task)
	}
	return nil
}

func (m *MockTaskRepository) Delete(id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(id)
	}
	return nil
}

func (m *MockTaskRepository) GetChildren(parentID string) ([]models.Task, error) {
	if m.getChildrenFunc != nil {
		return m.getChildrenFunc(parentID)
	}
	return nil, nil
}

func (m *MockTaskRepository) GetRoot(taskID string) (*models.Task, error) {
	if m.getRootFunc != nil {
		return m.getRootFunc(taskID)
	}
	return nil, errors.New("not found")
}

func (m *MockTaskRepository) GetDescendants(idPrefix string) ([]models.Task, error) {
	if m.getDescendantsFunc != nil {
		return m.getDescendantsFunc(idPrefix)
	}
	return nil, nil
}

// MockTaskDepRepository is a mock implementation of TaskDepRepositoryInterface.
type MockTaskDepRepository struct {
	addDepFunc        func(*models.TaskDep) error
	removeDepFunc     func(string, string) error
	getBlockersFunc   func(string) ([]models.TaskDep, error)
	getDependents     func(string) ([]models.TaskDep, error)
	getAllForTaskFunc func(string) ([]models.TaskDep, error)
}

func (m *MockTaskDepRepository) AddDep(dep *models.TaskDep) error {
	if m.addDepFunc != nil {
		return m.addDepFunc(dep)
	}
	return nil
}

func (m *MockTaskDepRepository) RemoveDep(fromID, toID string) error {
	if m.removeDepFunc != nil {
		return m.removeDepFunc(fromID, toID)
	}
	return nil
}

func (m *MockTaskDepRepository) GetBlockers(taskID string) ([]models.TaskDep, error) {
	if m.getBlockersFunc != nil {
		return m.getBlockersFunc(taskID)
	}
	return nil, nil
}

func (m *MockTaskDepRepository) GetDependents(taskID string) ([]models.TaskDep, error) {
	if m.getDependents != nil {
		return m.getDependents(taskID)
	}
	return nil, nil
}

func (m *MockTaskDepRepository) GetAllForTask(taskID string) ([]models.TaskDep, error) {
	if m.getAllForTaskFunc != nil {
		return m.getAllForTaskFunc(taskID)
	}
	return nil, nil
}

func (m *MockTaskDepRepository) GetDepsAmongTasks(_ []string) ([]models.TaskDep, error) {
	return nil, nil
}

// MockTimeEventRepository is a mock implementation of TimeEventRepositoryInterface.
type MockTimeEventRepository struct {
	createFunc func(*models.TimeEvent) error
	Events     []models.TimeEvent // captured events for assertions
}

func (m *MockTimeEventRepository) Create(event *models.TimeEvent) error {
	m.Events = append(m.Events, *event)
	if m.createFunc != nil {
		return m.createFunc(event)
	}
	return nil
}

// newTestTaskService creates a TaskService with the given mocks.
func newTestTaskService(taskRepo *MockTaskRepository, taskDepRepo *MockTaskDepRepository) *TaskService {
	return NewTaskService(taskRepo, taskDepRepo, &MockTimeEventRepository{})
}

// --- StartTask Tests ---

func TestStartTask_Success(t *testing.T) {
	// Arrange
	taskID := "job-abc.1"
	userID := uuid.New()

	task := &models.Task{
		ID:     taskID,
		Status: models.TaskStatusOpen,
		Title:  "Cut mortises",
	}

	var updatedTask *models.Task
	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			if id == taskID {
				return task, nil
			}
			return nil, errors.New("task not found")
		},
		updateFunc: func(t *models.Task) error {
			updatedTask = t
			return nil
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.StartTask(taskID, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.TaskStatusActive, result.Status)
	assert.NotNil(t, result.StartedAt)
	assert.WithinDuration(t, time.Now(), *result.StartedAt, 2*time.Second)
	assert.WithinDuration(t, time.Now(), result.UpdatedAt, 2*time.Second)

	// Verify update was called
	assert.NotNil(t, updatedTask)
	assert.Equal(t, models.TaskStatusActive, updatedTask.Status)
}

func TestStartTask_TaskNotFound(t *testing.T) {
	// Arrange
	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			return nil, errors.New("task not found")
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.StartTask("nonexistent", uuid.New())

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "task not found")
}

func TestStartTask_NotOpenStatus(t *testing.T) {
	// Arrange
	taskID := "job-abc.1"
	userID := uuid.New()

	// Test each non-open status
	statuses := []models.TaskStatus{
		models.TaskStatusActive,
		models.TaskStatusPaused,
		models.TaskStatusDone,
		models.TaskStatusSkipped,
		models.TaskStatusCancelled,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			task := &models.Task{
				ID:     taskID,
				Status: status,
				Title:  "Some task",
			}

			mockRepo := &MockTaskRepository{
				getByIDFunc: func(id string) (*models.Task, error) {
					return task, nil
				},
			}

			svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

			// Act
			result, err := svc.StartTask(taskID, userID)

			// Assert
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "cannot be started")
			assert.Contains(t, err.Error(), string(status))
		})
	}
}

func TestStartTask_UpdateFails(t *testing.T) {
	// Arrange
	taskID := "job-abc.1"
	userID := uuid.New()

	task := &models.Task{
		ID:     taskID,
		Status: models.TaskStatusOpen,
		Title:  "Some task",
	}

	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			return task, nil
		},
		updateFunc: func(t *models.Task) error {
			return errors.New("database error")
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.StartTask(taskID, userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "database error")
}

func TestStartTask_PreservesExistingStartedAt(t *testing.T) {
	// Arrange: task that was started before, somehow set back to open
	taskID := "job-abc.1"
	userID := uuid.New()
	originalStartedAt := time.Now().Add(-1 * time.Hour)

	task := &models.Task{
		ID:        taskID,
		Status:    models.TaskStatusOpen,
		StartedAt: &originalStartedAt,
		Title:     "Restarted task",
	}

	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			return task, nil
		},
		updateFunc: func(t *models.Task) error {
			return nil
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.StartTask(taskID, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.TaskStatusActive, result.Status)
	// StartedAt should NOT be overwritten
	assert.Equal(t, originalStartedAt, *result.StartedAt)
}

// --- PauseTask Tests ---

func TestPauseTask_Success(t *testing.T) {
	// Arrange
	taskID := "job-abc.3"
	userID := uuid.New()
	startedAt := time.Now().Add(-10 * time.Minute)

	task := &models.Task{
		ID:        taskID,
		Status:    models.TaskStatusActive,
		StartedAt: &startedAt,
		Title:     "Cut mortises",
	}

	var updatedTask *models.Task
	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			if id == taskID {
				return task, nil
			}
			return nil, errors.New("task not found")
		},
		updateFunc: func(t *models.Task) error {
			updatedTask = t
			return nil
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.PauseTask(taskID, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.TaskStatusPaused, result.Status)
	assert.NotNil(t, result.PausedAt)
	assert.WithinDuration(t, time.Now(), *result.PausedAt, 2*time.Second)
	assert.WithinDuration(t, time.Now(), result.UpdatedAt, 2*time.Second)

	// Verify update was called
	assert.NotNil(t, updatedTask)
	assert.Equal(t, models.TaskStatusPaused, updatedTask.Status)
}

func TestPauseTask_TaskNotFound(t *testing.T) {
	// Arrange
	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			return nil, errors.New("task not found")
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.PauseTask("nonexistent", uuid.New())

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "task not found")
}

func TestPauseTask_NotActiveStatus(t *testing.T) {
	// Arrange
	taskID := "job-abc.3"
	userID := uuid.New()

	// Test each non-active status
	statuses := []models.TaskStatus{
		models.TaskStatusOpen,
		models.TaskStatusPaused,
		models.TaskStatusDone,
		models.TaskStatusSkipped,
		models.TaskStatusCancelled,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			task := &models.Task{
				ID:     taskID,
				Status: status,
				Title:  "Some task",
			}

			mockRepo := &MockTaskRepository{
				getByIDFunc: func(id string) (*models.Task, error) {
					return task, nil
				},
			}

			svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

			// Act
			result, err := svc.PauseTask(taskID, userID)

			// Assert
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "cannot be paused")
			assert.Contains(t, err.Error(), string(status))
		})
	}
}

// --- ResumeTask Tests ---

func TestResumeTask_Success(t *testing.T) {
	// Arrange
	taskID := "job-abc.3"
	userID := uuid.New()
	pausedAt := time.Now().Add(-5 * time.Minute)
	startedAt := time.Now().Add(-15 * time.Minute)

	task := &models.Task{
		ID:        taskID,
		Status:    models.TaskStatusPaused,
		StartedAt: &startedAt,
		PausedAt:  &pausedAt,
		Title:     "Cut mortises",
	}

	var updatedTask *models.Task
	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			if id == taskID {
				return task, nil
			}
			return nil, errors.New("task not found")
		},
		updateFunc: func(t *models.Task) error {
			updatedTask = t
			return nil
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.ResumeTask(taskID, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.TaskStatusActive, result.Status)
	assert.Nil(t, result.PausedAt, "PausedAt should be cleared on resume")
	assert.NotNil(t, result.StartedAt, "StartedAt should be preserved")
	assert.WithinDuration(t, time.Now(), result.UpdatedAt, 2*time.Second)

	// Verify update was called
	assert.NotNil(t, updatedTask)
	assert.Equal(t, models.TaskStatusActive, updatedTask.Status)
	assert.Nil(t, updatedTask.PausedAt)
}

func TestResumeTask_TaskNotFound(t *testing.T) {
	// Arrange
	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			return nil, errors.New("task not found")
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.ResumeTask("nonexistent", uuid.New())

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "task not found")
}

func TestResumeTask_NotPausedStatus(t *testing.T) {
	// Arrange
	taskID := "job-abc.3"
	userID := uuid.New()

	// Test each non-paused status
	statuses := []models.TaskStatus{
		models.TaskStatusOpen,
		models.TaskStatusActive,
		models.TaskStatusDone,
		models.TaskStatusSkipped,
		models.TaskStatusCancelled,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			task := &models.Task{
				ID:     taskID,
				Status: status,
				Title:  "Some task",
			}

			mockRepo := &MockTaskRepository{
				getByIDFunc: func(id string) (*models.Task, error) {
					return task, nil
				},
			}

			svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

			// Act
			result, err := svc.ResumeTask(taskID, userID)

			// Assert
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "cannot be resumed")
			assert.Contains(t, err.Error(), string(status))
		})
	}
}

func TestResumeTask_UpdateFails(t *testing.T) {
	// Arrange
	taskID := "job-abc.3"
	userID := uuid.New()
	pausedAt := time.Now().Add(-5 * time.Minute)

	task := &models.Task{
		ID:       taskID,
		Status:   models.TaskStatusPaused,
		PausedAt: &pausedAt,
		Title:    "Some task",
	}

	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			return task, nil
		},
		updateFunc: func(t *models.Task) error {
			return errors.New("database error")
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.ResumeTask(taskID, userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "database error")
}

// --- SkipTask Tests ---

func TestSkipTask_Success_OpenTask(t *testing.T) {
	// Arrange — skip an open, unassigned task
	taskID := "job-abc.3"
	userID := uuid.New()

	task := &models.Task{
		ID:     taskID,
		Status: models.TaskStatusOpen,
		Title:  "Cut mortises",
	}

	var updatedTask *models.Task
	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			if id == taskID {
				return task, nil
			}
			return nil, errors.New("task not found")
		},
		updateFunc: func(t *models.Task) error {
			updatedTask = t
			return nil
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.SkipTask(taskID, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.TaskStatusSkipped, result.Status)
	assert.NotNil(t, result.CompletedAt)
	assert.WithinDuration(t, time.Now(), *result.CompletedAt, 2*time.Second)
	assert.WithinDuration(t, time.Now(), result.UpdatedAt, 2*time.Second)

	assert.NotNil(t, updatedTask)
	assert.Equal(t, models.TaskStatusSkipped, updatedTask.Status)
}

func TestSkipTask_Success_ActiveTask(t *testing.T) {
	// Arrange — skip an active task assigned to the user
	taskID := "job-abc.3"
	userID := uuid.New()
	startedAt := time.Now().Add(-10 * time.Minute)

	task := &models.Task{
		ID:           taskID,
		Status:       models.TaskStatusActive,
		AssignedToID: &userID,
		StartedAt:    &startedAt,
		Title:        "Cut mortises",
	}

	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			if id == taskID {
				return task, nil
			}
			return nil, errors.New("task not found")
		},
		updateFunc: func(t *models.Task) error {
			return nil
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.SkipTask(taskID, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.TaskStatusSkipped, result.Status)
	assert.NotNil(t, result.CompletedAt)
}

func TestSkipTask_Success_PausedTask(t *testing.T) {
	// Arrange — skip a paused task assigned to the user
	taskID := "job-abc.3"
	userID := uuid.New()
	pausedAt := time.Now().Add(-5 * time.Minute)

	task := &models.Task{
		ID:           taskID,
		Status:       models.TaskStatusPaused,
		AssignedToID: &userID,
		PausedAt:     &pausedAt,
		Title:        "Cut mortises",
	}

	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			if id == taskID {
				return task, nil
			}
			return nil, errors.New("task not found")
		},
		updateFunc: func(t *models.Task) error {
			return nil
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.SkipTask(taskID, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.TaskStatusSkipped, result.Status)
}

func TestSkipTask_TaskNotFound(t *testing.T) {
	// Arrange
	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			return nil, errors.New("task not found")
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.SkipTask("nonexistent", uuid.New())

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "task not found")
}

func TestSkipTask_AlreadyTerminal(t *testing.T) {
	// Arrange — test each terminal status
	taskID := "job-abc.3"
	userID := uuid.New()

	terminalStatuses := []models.TaskStatus{
		models.TaskStatusDone,
		models.TaskStatusSkipped,
		models.TaskStatusCancelled,
	}

	for _, status := range terminalStatuses {
		t.Run(string(status), func(t *testing.T) {
			task := &models.Task{
				ID:     taskID,
				Status: status,
				Title:  "Some task",
			}

			mockRepo := &MockTaskRepository{
				getByIDFunc: func(id string) (*models.Task, error) {
					return task, nil
				},
			}

			svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

			// Act
			result, err := svc.SkipTask(taskID, userID)

			// Assert
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "cannot be skipped")
			assert.Contains(t, err.Error(), "already terminal")
		})
	}
}

func TestSkipTask_TriggersParentAutoComplete(t *testing.T) {
	// Arrange — skip the last non-terminal child to trigger parent auto-complete
	parentID := "job-abc"
	taskID := "job-abc.2"
	userID := uuid.New()

	parent := &models.Task{
		ID:     parentID,
		Status: models.TaskStatusActive,
		Title:  "Parent job",
	}

	task := &models.Task{
		ID:       taskID,
		Status:   models.TaskStatusOpen,
		ParentID: &parentID,
		Title:    "Second step",
	}

	sibling := models.Task{
		ID:       "job-abc.1",
		Status:   models.TaskStatusDone,
		ParentID: &parentID,
		Title:    "First step",
	}

	updatedIDs := map[string]bool{}
	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			switch id {
			case taskID:
				return task, nil
			case parentID:
				return parent, nil
			}
			return nil, errors.New("not found")
		},
		getChildrenFunc: func(pid string) ([]models.Task, error) {
			if pid == parentID {
				// Return the skipped task (simulating post-update state) and the done sibling
				skippedTask := *task
				skippedTask.Status = models.TaskStatusSkipped
				return []models.Task{skippedTask, sibling}, nil
			}
			return nil, nil
		},
		updateFunc: func(t *models.Task) error {
			updatedIDs[t.ID] = true
			return nil
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.SkipTask(taskID, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.TaskStatusSkipped, result.Status)
	assert.True(t, updatedIDs[taskID], "task should have been updated")
	assert.True(t, updatedIDs[parentID], "parent should have been auto-completed")
}

func TestSkipTask_UpdateFails(t *testing.T) {
	// Arrange
	taskID := "job-abc.3"
	userID := uuid.New()

	task := &models.Task{
		ID:     taskID,
		Status: models.TaskStatusOpen,
		Title:  "Some task",
	}

	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			return task, nil
		},
		updateFunc: func(t *models.Task) error {
			return errors.New("database error")
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.SkipTask(taskID, userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "database error")
}

// --- CompleteTask Tests ---

func TestCompleteTask_Success(t *testing.T) {
	// Arrange
	taskID := "job-abc.3"
	userID := uuid.New()

	startedAt := time.Now().Add(-10 * time.Minute)
	task := &models.Task{
		ID:        taskID,
		Status:    models.TaskStatusActive,
		StartedAt: &startedAt,
		Title:     "Cut mortises",
	}

	var updatedTask *models.Task
	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			if id == taskID {
				return task, nil
			}
			return nil, errors.New("task not found")
		},
		updateFunc: func(t *models.Task) error {
			updatedTask = t
			return nil
		},
	}

	mockDepRepo := &MockTaskDepRepository{
		getBlockersFunc: func(id string) ([]models.TaskDep, error) {
			return nil, nil // no deps
		},
	}

	svc := newTestTaskService(mockRepo, mockDepRepo)

	// Act
	result, err := svc.CompleteTask(taskID, userID, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.TaskStatusDone, result.Task.Status)
	assert.NotNil(t, result.Task.CompletedAt)
	assert.WithinDuration(t, time.Now(), *result.Task.CompletedAt, 2*time.Second)
	assert.WithinDuration(t, time.Now(), result.Task.UpdatedAt, 2*time.Second)

	// Verify update was called
	assert.NotNil(t, updatedTask)
	assert.Equal(t, models.TaskStatusDone, updatedTask.Status)
}

func TestCompleteTask_TaskNotFound(t *testing.T) {
	// Arrange
	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			return nil, errors.New("task not found")
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.CompleteTask("nonexistent", uuid.New(), nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "task not found")
}

func TestCompleteTask_NotActiveStatus(t *testing.T) {
	// Arrange
	taskID := "job-abc.3"
	userID := uuid.New()

	statuses := []models.TaskStatus{
		models.TaskStatusOpen,
		models.TaskStatusPaused,
		models.TaskStatusDone,
		models.TaskStatusSkipped,
		models.TaskStatusCancelled,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			task := &models.Task{
				ID:     taskID,
				Status: status,
				Title:  "Some task",
			}

			mockRepo := &MockTaskRepository{
				getByIDFunc: func(id string) (*models.Task, error) {
					return task, nil
				},
			}

			svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

			// Act
			result, err := svc.CompleteTask(taskID, userID, nil)

			// Assert
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "cannot be completed")
			assert.Contains(t, err.Error(), string(status))
		})
	}
}

func TestCompleteTask_UnresolvedBlockingDep(t *testing.T) {
	// Arrange
	taskID := "job-abc.3"
	blockerID := "job-abc.2"
	userID := uuid.New()

	task := &models.Task{
		ID:           taskID,
		Status:       models.TaskStatusActive,
		AssignedToID: &userID,
		Title:        "Cut mortises",
	}

	blockerTask := &models.Task{
		ID:     blockerID,
		Status: models.TaskStatusOpen, // not terminal
		Title:  "Select lumber",
	}

	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			switch id {
			case taskID:
				return task, nil
			case blockerID:
				return blockerTask, nil
			}
			return nil, errors.New("task not found")
		},
	}

	mockDepRepo := &MockTaskDepRepository{
		getBlockersFunc: func(id string) ([]models.TaskDep, error) {
			if id == taskID {
				return []models.TaskDep{
					{
						FromTaskID: taskID,
						ToTaskID:   blockerID,
						Type:       models.DepTypeBlocks,
					},
				}, nil
			}
			return nil, nil
		},
	}

	svc := newTestTaskService(mockRepo, mockDepRepo)

	// Act
	result, err := svc.CompleteTask(taskID, userID, nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "blocked by task")
	assert.Contains(t, err.Error(), blockerID)
}

func TestCompleteTask_ResolvedBlockingDeps(t *testing.T) {
	// Test all terminal statuses resolve blocking deps.
	taskID := "job-abc.3"
	userID := uuid.New()

	terminalStatuses := []models.TaskStatus{
		models.TaskStatusDone,
		models.TaskStatusSkipped,
		models.TaskStatusCancelled,
	}

	for _, blockerStatus := range terminalStatuses {
		t.Run(string(blockerStatus), func(t *testing.T) {
			blockerID := "job-abc.2"

			task := &models.Task{
				ID:           taskID,
				Status:       models.TaskStatusActive,
				AssignedToID: &userID,
				Title:        "Cut mortises",
			}

			blockerTask := &models.Task{
				ID:     blockerID,
				Status: blockerStatus,
				Title:  "Select lumber",
			}

			mockRepo := &MockTaskRepository{
				getByIDFunc: func(id string) (*models.Task, error) {
					switch id {
					case taskID:
						return task, nil
					case blockerID:
						return blockerTask, nil
					}
					return nil, errors.New("task not found")
				},
				updateFunc: func(t *models.Task) error {
					return nil
				},
			}

			mockDepRepo := &MockTaskDepRepository{
				getBlockersFunc: func(id string) ([]models.TaskDep, error) {
					if id == taskID {
						return []models.TaskDep{
							{
								FromTaskID: taskID,
								ToTaskID:   blockerID,
								Type:       models.DepTypeBlocks,
							},
						}, nil
					}
					return nil, nil
				},
			}

			svc := newTestTaskService(mockRepo, mockDepRepo)

			// Act
			result, err := svc.CompleteTask(taskID, userID, nil)

			// Assert
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, models.TaskStatusDone, result.Task.Status)
		})
	}
}

func TestCompleteTask_NonBlockingDepsIgnored(t *testing.T) {
	// waits_for and related deps should not prevent completion.
	taskID := "job-abc.3"
	otherID := "job-abc.2"
	userID := uuid.New()

	task := &models.Task{
		ID:           taskID,
		Status:       models.TaskStatusActive,
		AssignedToID: &userID,
		Title:        "Cut mortises",
	}

	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			if id == taskID {
				return task, nil
			}
			return nil, errors.New("task not found")
		},
		updateFunc: func(t *models.Task) error {
			return nil
		},
	}

	mockDepRepo := &MockTaskDepRepository{
		getBlockersFunc: func(id string) ([]models.TaskDep, error) {
			if id == taskID {
				return []models.TaskDep{
					{
						FromTaskID: taskID,
						ToTaskID:   otherID,
						Type:       models.DepTypeWaitsFor,
					},
					{
						FromTaskID: taskID,
						ToTaskID:   otherID,
						Type:       models.DepTypeRelated,
					},
				}, nil
			}
			return nil, nil
		},
	}

	svc := newTestTaskService(mockRepo, mockDepRepo)

	// Act
	result, err := svc.CompleteTask(taskID, userID, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.TaskStatusDone, result.Task.Status)
}

func TestCompleteTask_AutoCompletesParent(t *testing.T) {
	// When all siblings are done, parent should be auto-completed.
	taskID := "job-abc.3"
	parentID := "job-abc"
	userID := uuid.New()

	task := &models.Task{
		ID:           taskID,
		ParentID:     &parentID,
		Status:       models.TaskStatusActive,
		AssignedToID: &userID,
		Title:        "Last child task",
	}

	parentTask := &models.Task{
		ID:     parentID,
		Status: models.TaskStatusActive,
		Title:  "Parent job",
	}

	siblingDone := models.Task{
		ID:       "job-abc.1",
		ParentID: &parentID,
		Status:   models.TaskStatusDone,
		Title:    "First child",
	}

	siblingSkipped := models.Task{
		ID:       "job-abc.2",
		ParentID: &parentID,
		Status:   models.TaskStatusSkipped,
		Title:    "Second child",
	}

	var updatedTasks []*models.Task
	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			switch id {
			case taskID:
				return task, nil
			case parentID:
				return parentTask, nil
			}
			return nil, errors.New("task not found")
		},
		updateFunc: func(t *models.Task) error {
			updatedTasks = append(updatedTasks, t)
			return nil
		},
		getChildrenFunc: func(pid string) ([]models.Task, error) {
			if pid == parentID {
				// Return children with the completing task already marked done
				// (since it was updated in-memory before this call).
				return []models.Task{
					siblingDone,
					siblingSkipped,
					{
						ID:       taskID,
						ParentID: &parentID,
						Status:   models.TaskStatusDone, // already updated
						Title:    "Last child task",
					},
				}, nil
			}
			return nil, nil
		},
	}

	mockDepRepo := &MockTaskDepRepository{
		getBlockersFunc: func(id string) ([]models.TaskDep, error) {
			return nil, nil
		},
	}

	svc := newTestTaskService(mockRepo, mockDepRepo)

	// Act
	result, err := svc.CompleteTask(taskID, userID, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.TaskStatusDone, result.Task.Status)

	// Verify parent was also updated
	assert.Len(t, updatedTasks, 2) // task + parent
	assert.Equal(t, models.TaskStatusDone, updatedTasks[1].Status)
	assert.NotNil(t, updatedTasks[1].CompletedAt)
}

func TestCompleteTask_ParentNotCompletedWhenChildrenRemain(t *testing.T) {
	// When some siblings are not done, parent should NOT be auto-completed.
	taskID := "job-abc.3"
	parentID := "job-abc"
	userID := uuid.New()

	task := &models.Task{
		ID:           taskID,
		ParentID:     &parentID,
		Status:       models.TaskStatusActive,
		AssignedToID: &userID,
		Title:        "A child task",
	}

	parentTask := &models.Task{
		ID:     parentID,
		Status: models.TaskStatusActive,
		Title:  "Parent job",
	}

	siblingOpen := models.Task{
		ID:       "job-abc.1",
		ParentID: &parentID,
		Status:   models.TaskStatusOpen,
		Title:    "Still open",
	}

	updateCount := 0
	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			switch id {
			case taskID:
				return task, nil
			case parentID:
				return parentTask, nil
			}
			return nil, errors.New("task not found")
		},
		updateFunc: func(t *models.Task) error {
			updateCount++
			return nil
		},
		getChildrenFunc: func(pid string) ([]models.Task, error) {
			if pid == parentID {
				return []models.Task{
					siblingOpen,
					{
						ID:       taskID,
						ParentID: &parentID,
						Status:   models.TaskStatusDone,
						Title:    "A child task",
					},
				}, nil
			}
			return nil, nil
		},
	}

	mockDepRepo := &MockTaskDepRepository{
		getBlockersFunc: func(id string) ([]models.TaskDep, error) {
			return nil, nil
		},
	}

	svc := newTestTaskService(mockRepo, mockDepRepo)

	// Act
	result, err := svc.CompleteTask(taskID, userID, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.TaskStatusDone, result.Task.Status)

	// Only the task itself should be updated, not the parent
	assert.Equal(t, 1, updateCount)
}

func TestCompleteTask_UpdateFails(t *testing.T) {
	// Arrange
	taskID := "job-abc.3"
	userID := uuid.New()

	task := &models.Task{
		ID:           taskID,
		Status:       models.TaskStatusActive,
		AssignedToID: &userID,
		Title:        "Some task",
	}

	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			return task, nil
		},
		updateFunc: func(t *models.Task) error {
			return errors.New("database error")
		},
	}

	mockDepRepo := &MockTaskDepRepository{
		getBlockersFunc: func(id string) ([]models.TaskDep, error) {
			return nil, nil
		},
	}

	svc := newTestTaskService(mockRepo, mockDepRepo)

	// Act
	result, err := svc.CompleteTask(taskID, userID, nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "database error")
}

func TestCompleteTask_GetBlockersFails(t *testing.T) {
	// Arrange
	taskID := "job-abc.3"
	userID := uuid.New()

	task := &models.Task{
		ID:           taskID,
		Status:       models.TaskStatusActive,
		AssignedToID: &userID,
		Title:        "Some task",
	}

	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			return task, nil
		},
	}

	mockDepRepo := &MockTaskDepRepository{
		getBlockersFunc: func(id string) ([]models.TaskDep, error) {
			return nil, errors.New("dep query failed")
		},
	}

	svc := newTestTaskService(mockRepo, mockDepRepo)

	// Act
	result, err := svc.CompleteTask(taskID, userID, nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to check dependencies")
}

func TestCompleteTask_BlockerLookupFails(t *testing.T) {
	// Arrange
	taskID := "job-abc.3"
	userID := uuid.New()

	task := &models.Task{
		ID:           taskID,
		Status:       models.TaskStatusActive,
		AssignedToID: &userID,
		Title:        "Some task",
	}

	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			if id == taskID {
				return task, nil
			}
			return nil, errors.New("blocker not found")
		},
	}

	mockDepRepo := &MockTaskDepRepository{
		getBlockersFunc: func(id string) ([]models.TaskDep, error) {
			return []models.TaskDep{
				{
					FromTaskID: taskID,
					ToTaskID:   "missing-task",
					Type:       models.DepTypeBlocks,
				},
			}, nil
		},
	}

	svc := newTestTaskService(mockRepo, mockDepRepo)

	// Act
	result, err := svc.CompleteTask(taskID, userID, nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to look up blocking task")
}

func TestCompleteTask_NoParent(t *testing.T) {
	// Task with no parent should complete without attempting parent auto-complete.
	taskID := "job-abc"
	userID := uuid.New()

	task := &models.Task{
		ID:           taskID,
		ParentID:     nil,
		Status:       models.TaskStatusActive,
		AssignedToID: &userID,
		Title:        "Root task",
	}

	updateCount := 0
	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			if id == taskID {
				return task, nil
			}
			return nil, errors.New("task not found")
		},
		updateFunc: func(t *models.Task) error {
			updateCount++
			return nil
		},
	}

	mockDepRepo := &MockTaskDepRepository{
		getBlockersFunc: func(id string) ([]models.TaskDep, error) {
			return nil, nil
		},
	}

	svc := newTestTaskService(mockRepo, mockDepRepo)

	// Act
	result, err := svc.CompleteTask(taskID, userID, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.TaskStatusDone, result.Task.Status)
	assert.Equal(t, 1, updateCount) // only the task itself
}

func TestCompleteTask_ParentAlreadyDone(t *testing.T) {
	// If parent is already done, don't try to auto-complete it again.
	taskID := "job-abc.3"
	parentID := "job-abc"
	userID := uuid.New()

	task := &models.Task{
		ID:           taskID,
		ParentID:     &parentID,
		Status:       models.TaskStatusActive,
		AssignedToID: &userID,
		Title:        "A child task",
	}

	parentTask := &models.Task{
		ID:     parentID,
		Status: models.TaskStatusDone, // already done
		Title:  "Parent job",
	}

	updateCount := 0
	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			switch id {
			case taskID:
				return task, nil
			case parentID:
				return parentTask, nil
			}
			return nil, errors.New("task not found")
		},
		updateFunc: func(t *models.Task) error {
			updateCount++
			return nil
		},
	}

	mockDepRepo := &MockTaskDepRepository{
		getBlockersFunc: func(id string) ([]models.TaskDep, error) {
			return nil, nil
		},
	}

	svc := newTestTaskService(mockRepo, mockDepRepo)

	// Act
	result, err := svc.CompleteTask(taskID, userID, nil)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.TaskStatusDone, result.Task.Status)
	// Only the task itself should be updated, not the already-done parent
	assert.Equal(t, 1, updateCount)
}

func TestPauseTask_UpdateFails(t *testing.T) {
	// Arrange
	taskID := "job-abc.3"
	userID := uuid.New()

	task := &models.Task{
		ID:           taskID,
		Status:       models.TaskStatusActive,
		AssignedToID: &userID,
		Title:        "Some task",
	}

	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			return task, nil
		},
		updateFunc: func(t *models.Task) error {
			return errors.New("database error")
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.PauseTask(taskID, userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "database error")
}

// --- AddChildTask Tests ---

func TestAddChildTask_Success_FirstChild(t *testing.T) {
	// Arrange
	parentID := "job-abc"
	userID := uuid.New()
	spaceID := uuid.New()
	desc := "Cut all mortise joints"

	parentTask := &models.Task{
		ID:       parentID,
		SpaceID:  spaceID,
		Status:   models.TaskStatusActive,
		Title:    "Build dining table",
		Priority: 2,
	}

	var createdTask *models.Task
	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			if id == parentID {
				return parentTask, nil
			}
			return nil, errors.New("task not found")
		},
		getChildrenFunc: func(pid string) ([]models.Task, error) {
			if pid == parentID {
				return []models.Task{}, nil // no existing children
			}
			return nil, nil
		},
		createFunc: func(task *models.Task) error {
			createdTask = task
			return nil
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.AddChildTask(parentID, &dtos.AddChildTaskRequest{Title: "Cut mortises", Description: &desc}, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "job-abc.1", result.ID)
	assert.Equal(t, &parentID, result.ParentID)
	assert.Equal(t, spaceID, result.SpaceID)
	assert.Equal(t, userID, result.CreatedByID)
	assert.Equal(t, models.TaskTypeTask, result.Type)
	assert.Equal(t, models.TaskStatusOpen, result.Status)
	assert.Equal(t, "Cut mortises", result.Title)
	assert.Equal(t, &desc, result.Description)
	assert.Equal(t, 2, result.Priority) // defaults to P2 (Medium)
	assert.WithinDuration(t, time.Now(), result.CreatedAt, 2*time.Second)
	assert.WithinDuration(t, time.Now(), result.UpdatedAt, 2*time.Second)

	// Verify create was called with the right task
	assert.NotNil(t, createdTask)
	assert.Equal(t, "job-abc.1", createdTask.ID)
}

func TestAddChildTask_Success_SequentialIDs(t *testing.T) {
	// When parent already has children, the next child gets the next sequence number.
	parentID := "job-abc"
	userID := uuid.New()
	spaceID := uuid.New()

	parentTask := &models.Task{
		ID:      parentID,
		SpaceID: spaceID,
		Status:  models.TaskStatusActive,
		Title:   "Build dining table",
	}

	existingChildren := []models.Task{
		{ID: "job-abc.1", ParentID: &parentID, Title: "Step 1"},
		{ID: "job-abc.2", ParentID: &parentID, Title: "Step 2"},
		{ID: "job-abc.3", ParentID: &parentID, Title: "Step 3"},
	}

	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			if id == parentID {
				return parentTask, nil
			}
			return nil, errors.New("task not found")
		},
		getChildrenFunc: func(pid string) ([]models.Task, error) {
			if pid == parentID {
				return existingChildren, nil
			}
			return nil, nil
		},
		createFunc: func(task *models.Task) error {
			return nil
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.AddChildTask(parentID, &dtos.AddChildTaskRequest{Title: "Step 4"}, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "job-abc.4", result.ID)
}

func TestAddChildTask_Success_NestedHierarchy(t *testing.T) {
	// Adding a child to a child task produces deeper hierarchical IDs.
	parentID := "job-abc.2"
	userID := uuid.New()
	spaceID := uuid.New()

	parentTask := &models.Task{
		ID:      parentID,
		SpaceID: spaceID,
		Status:  models.TaskStatusActive,
		Title:   "Assemble frame",
	}

	existingChildren := []models.Task{
		{ID: "job-abc.2.1", ParentID: &parentID, Title: "Sub-step 1"},
	}

	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			if id == parentID {
				return parentTask, nil
			}
			return nil, errors.New("task not found")
		},
		getChildrenFunc: func(pid string) ([]models.Task, error) {
			if pid == parentID {
				return existingChildren, nil
			}
			return nil, nil
		},
		createFunc: func(task *models.Task) error {
			return nil
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.AddChildTask(parentID, &dtos.AddChildTaskRequest{Title: "Sub-step 2"}, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "job-abc.2.2", result.ID)
}

func TestAddChildTask_Success_NilDescription(t *testing.T) {
	// Description is optional — nil should work fine.
	parentID := "job-abc"
	userID := uuid.New()
	spaceID := uuid.New()

	parentTask := &models.Task{
		ID:      parentID,
		SpaceID: spaceID,
		Title:   "Build dining table",
	}

	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			if id == parentID {
				return parentTask, nil
			}
			return nil, errors.New("task not found")
		},
		getChildrenFunc: func(pid string) ([]models.Task, error) {
			return []models.Task{}, nil
		},
		createFunc: func(task *models.Task) error {
			return nil
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.AddChildTask(parentID, &dtos.AddChildTaskRequest{Title: "Quick step"}, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Nil(t, result.Description)
}

func TestAddChildTask_EmptyTitle(t *testing.T) {
	svc := newTestTaskService(&MockTaskRepository{}, &MockTaskDepRepository{})

	// Act
	result, err := svc.AddChildTask("job-abc", &dtos.AddChildTaskRequest{Title: ""}, uuid.New())

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "title is required")
}

func TestAddChildTask_ParentNotFound(t *testing.T) {
	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			return nil, errors.New("task not found")
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.AddChildTask("nonexistent", &dtos.AddChildTaskRequest{Title: "Some step"}, uuid.New())

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "parent task")
	assert.Contains(t, err.Error(), "not found")
}

func TestAddChildTask_GetChildrenFails(t *testing.T) {
	parentID := "job-abc"

	parentTask := &models.Task{
		ID:      parentID,
		SpaceID: uuid.New(),
		Title:   "Build dining table",
	}

	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			if id == parentID {
				return parentTask, nil
			}
			return nil, errors.New("task not found")
		},
		getChildrenFunc: func(pid string) ([]models.Task, error) {
			return nil, errors.New("database error")
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.AddChildTask(parentID, &dtos.AddChildTaskRequest{Title: "Some step"}, uuid.New())

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to get children")
}

func TestAddChildTask_CreateFails(t *testing.T) {
	parentID := "job-abc"

	parentTask := &models.Task{
		ID:      parentID,
		SpaceID: uuid.New(),
		Title:   "Build dining table",
	}

	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			if id == parentID {
				return parentTask, nil
			}
			return nil, errors.New("task not found")
		},
		getChildrenFunc: func(pid string) ([]models.Task, error) {
			return []models.Task{}, nil
		},
		createFunc: func(task *models.Task) error {
			return errors.New("unique constraint violation")
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.AddChildTask(parentID, &dtos.AddChildTaskRequest{Title: "Some step"}, uuid.New())

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create child task")
}
