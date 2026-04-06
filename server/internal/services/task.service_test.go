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

// --- CompleteTask Tests ---

func TestCompleteTask_Success(t *testing.T) {
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

	mockDepRepo := &MockTaskDepRepository{
		getDependents: func(id string) ([]models.TaskDep, error) {
			return nil, nil // no deps
		},
	}

	svc := newTestTaskService(mockRepo, mockDepRepo)

	// Act
	result, err := svc.CompleteTask(taskID, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.TaskStatusDone, result.Status)
	assert.NotNil(t, result.CompletedAt)
	assert.WithinDuration(t, time.Now(), *result.CompletedAt, 2*time.Second)
	assert.WithinDuration(t, time.Now(), result.UpdatedAt, 2*time.Second)

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
	result, err := svc.CompleteTask("nonexistent", uuid.New())

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
			result, err := svc.CompleteTask(taskID, userID)

			// Assert
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "cannot be completed")
			assert.Contains(t, err.Error(), string(status))
		})
	}
}

func TestCompleteTask_NotAssignedToUser(t *testing.T) {
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
	result, err := svc.CompleteTask(taskID, otherUserID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not assigned to user")
	assert.Contains(t, err.Error(), otherUserID.String())
}

func TestCompleteTask_NoAssignee(t *testing.T) {
	// Arrange
	taskID := "job-abc.3"
	userID := uuid.New()

	task := &models.Task{
		ID:           taskID,
		Status:       models.TaskStatusActive,
		AssignedToID: nil,
		Title:        "Some task",
	}

	mockRepo := &MockTaskRepository{
		getByIDFunc: func(id string) (*models.Task, error) {
			return task, nil
		},
	}

	svc := newTestTaskService(mockRepo, &MockTaskDepRepository{})

	// Act
	result, err := svc.CompleteTask(taskID, userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not assigned to user")
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
		getDependents: func(id string) ([]models.TaskDep, error) {
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
	result, err := svc.CompleteTask(taskID, userID)

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
				getDependents: func(id string) ([]models.TaskDep, error) {
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
			result, err := svc.CompleteTask(taskID, userID)

			// Assert
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, models.TaskStatusDone, result.Status)
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
		getDependents: func(id string) ([]models.TaskDep, error) {
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
	result, err := svc.CompleteTask(taskID, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.TaskStatusDone, result.Status)
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
		getDependents: func(id string) ([]models.TaskDep, error) {
			return nil, nil
		},
	}

	svc := newTestTaskService(mockRepo, mockDepRepo)

	// Act
	result, err := svc.CompleteTask(taskID, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.TaskStatusDone, result.Status)

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
		getDependents: func(id string) ([]models.TaskDep, error) {
			return nil, nil
		},
	}

	svc := newTestTaskService(mockRepo, mockDepRepo)

	// Act
	result, err := svc.CompleteTask(taskID, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.TaskStatusDone, result.Status)

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
		getDependents: func(id string) ([]models.TaskDep, error) {
			return nil, nil
		},
	}

	svc := newTestTaskService(mockRepo, mockDepRepo)

	// Act
	result, err := svc.CompleteTask(taskID, userID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "database error")
}

func TestCompleteTask_GetDependentsFails(t *testing.T) {
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
		getDependents: func(id string) ([]models.TaskDep, error) {
			return nil, errors.New("dep query failed")
		},
	}

	svc := newTestTaskService(mockRepo, mockDepRepo)

	// Act
	result, err := svc.CompleteTask(taskID, userID)

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
		getDependents: func(id string) ([]models.TaskDep, error) {
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
	result, err := svc.CompleteTask(taskID, userID)

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
		getDependents: func(id string) ([]models.TaskDep, error) {
			return nil, nil
		},
	}

	svc := newTestTaskService(mockRepo, mockDepRepo)

	// Act
	result, err := svc.CompleteTask(taskID, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.TaskStatusDone, result.Status)
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
		getDependents: func(id string) ([]models.TaskDep, error) {
			return nil, nil
		},
	}

	svc := newTestTaskService(mockRepo, mockDepRepo)

	// Act
	result, err := svc.CompleteTask(taskID, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.TaskStatusDone, result.Status)
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
