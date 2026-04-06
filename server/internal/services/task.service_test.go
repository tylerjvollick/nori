package services

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
)

// MockTaskRepository is a mock implementation of TaskRepositoryInterface.
type MockTaskRepository struct {
	createFunc      func(*models.Task) error
	getByIDFunc     func(string) (*models.Task, error)
	listFunc        func(repositories.TaskFilter) ([]models.Task, int64, error)
	updateFunc      func(*models.Task) error
	deleteFunc      func(string) error
	getChildrenFunc func(string) ([]models.Task, error)
	getRootFunc     func(string) (*models.Task, error)
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

// newTestTaskService creates a TaskService with the given mocks.
func newTestTaskService(taskRepo *MockTaskRepository, taskDepRepo *MockTaskDepRepository) *TaskService {
	return NewTaskService(taskRepo, taskDepRepo)
}

// --- ClaimTask Tests ---

func TestClaimTask_Success(t *testing.T) {
	// Arrange
	taskID := "job-abc.3"
	userID := uuid.New()
	spaceID := uuid.New()

	task := &models.Task{
		ID:      taskID,
		SpaceID: spaceID,
		Status:  models.TaskStatusOpen,
		Title:   "Cut mortises",
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
	result, err := svc.ClaimTask(taskID, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.TaskStatusActive, result.Status)
	assert.NotNil(t, result.AssignedToID)
	assert.Equal(t, userID, *result.AssignedToID)
	assert.NotNil(t, result.StartedAt)
	assert.WithinDuration(t, time.Now(), *result.StartedAt, 2*time.Second)
	assert.WithinDuration(t, time.Now(), result.UpdatedAt, 2*time.Second)

	// Verify update was called
	assert.NotNil(t, updatedTask)
	assert.Equal(t, models.TaskStatusActive, updatedTask.Status)
}

func TestClaimTask_TaskNotFound(t *testing.T) {
	// Arrange
	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			return nil, errors.New("task not found")
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.ClaimTask("nonexistent", uuid.New())

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "task not found")
}

func TestClaimTask_NotOpenStatus(t *testing.T) {
	// Arrange
	taskID := "job-abc.3"
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
			result, err := svc.ClaimTask(taskID, userID)

			// Assert
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "cannot be claimed")
			assert.Contains(t, err.Error(), string(status))
		})
	}
}

func TestClaimTask_AlreadyAssigned(t *testing.T) {
	// Arrange
	taskID := "job-abc.3"
	existingUserID := uuid.New()
	newUserID := uuid.New()

	task := &models.Task{
		ID:           taskID,
		Status:       models.TaskStatusOpen,
		AssignedToID: &existingUserID,
		Title:        "Some task",
	}

	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			return task, nil
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.ClaimTask(taskID, newUserID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "already assigned")
	assert.Contains(t, err.Error(), existingUserID.String())
}

func TestClaimTask_UpdateFails(t *testing.T) {
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
	result, err := svc.ClaimTask(taskID, userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "database error")
}

// --- PauseTask Tests ---

func TestPauseTask_Success(t *testing.T) {
	// Arrange
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
				ID:           taskID,
				Status:       status,
				AssignedToID: &userID,
				Title:        "Some task",
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

func TestPauseTask_NotAssignedToUser(t *testing.T) {
	// Arrange
	taskID := "job-abc.3"
	assignedUserID := uuid.New()
	otherUserID := uuid.New()

	task := &models.Task{
		ID:           taskID,
		Status:       models.TaskStatusActive,
		AssignedToID: &assignedUserID,
		Title:        "Some task",
	}

	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			return task, nil
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.PauseTask(taskID, otherUserID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not assigned to user")
	assert.Contains(t, err.Error(), otherUserID.String())
}

func TestPauseTask_NoAssignee(t *testing.T) {
	// Arrange
	taskID := "job-abc.3"
	userID := uuid.New()

	task := &models.Task{
		ID:           taskID,
		Status:       models.TaskStatusActive,
		AssignedToID: nil, // no assignee
		Title:        "Some task",
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
	assert.Contains(t, err.Error(), "not assigned to user")
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
