package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tylerjvollick/nori/internal/dbfactory"
	"github.com/tylerjvollick/nori/internal/dbtest"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
	"gorm.io/gorm"
)

// The services package TestMain (product.service_test.go) starts the shared
// PostgreSQL container these tests run against.

// newDBTaskService wires a TaskService against real repositories on the given
// per-test transaction, plus a space and user to own the test data.
func newDBTaskService(tx *gorm.DB) (*TaskService, models.Space, models.User) {
	svc := NewTaskService(
		repositories.NewTaskRepository(tx),
		repositories.NewTaskDepRepository(tx),
		repositories.NewTimeEventRepository(tx),
	)
	space := dbfactory.Space(tx)
	user := dbfactory.User(tx)
	return svc, space, user
}

func backlogStatusPtr() *models.TaskStatus {
	s := models.TaskStatusBacklog
	return &s
}

func TestCreateJob_WithBacklogStatus(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc, space, user := newDBTaskService(tx)

	job, err := svc.CreateTask(space.ID, user.ID, &dtos.CreateTaskRequest{
		Title:  "Backlog Job",
		Type:   models.TaskTypeJob,
		Status: backlogStatusPtr(),
	})
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatusBacklog, job.Status)

	// Verify the enum value persisted in PostgreSQL.
	var got models.Task
	require.NoError(t, tx.First(&got, "id = ?", job.ID).Error)
	assert.Equal(t, models.TaskStatusBacklog, got.Status)
}

func TestCreateTask_InvalidInitialStatus(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc, space, user := newDBTaskService(tx)

	done := models.TaskStatusDone
	_, err := svc.CreateTask(space.ID, user.ID, &dtos.CreateTaskRequest{
		Title:  "Bad Job",
		Type:   models.TaskTypeJob,
		Status: &done,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid initial status")
}

func TestBacklogJob_ScheduleThenStart(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc, space, user := newDBTaskService(tx)

	job, err := svc.CreateTask(space.ID, user.ID, &dtos.CreateTaskRequest{
		Title:  "Backlog Job",
		Type:   models.TaskTypeJob,
		Status: backlogStatusPtr(),
	})
	require.NoError(t, err)

	// backlog -> open (scheduled)
	job, err = svc.SetTaskStatus(job.ID, user.ID, models.TaskStatusOpen)
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatusOpen, job.Status)

	// open -> in_progress (started)
	job, err = svc.SetTaskStatus(job.ID, user.ID, models.TaskStatusInProgress)
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatusInProgress, job.Status)
	assert.NotNil(t, job.StartedAt, "StartedAt should be set on first transition to in_progress")

	// in_progress -> done
	job, err = svc.SetTaskStatus(job.ID, user.ID, models.TaskStatusDone)
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatusDone, job.Status)
}

func TestBacklogJob_ShortcutToInProgress(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc, space, user := newDBTaskService(tx)

	job, err := svc.CreateTask(space.ID, user.ID, &dtos.CreateTaskRequest{
		Title:  "Backlog Job",
		Type:   models.TaskTypeJob,
		Status: backlogStatusPtr(),
	})
	require.NoError(t, err)

	// backlog -> in_progress is an allowed shortcut.
	job, err = svc.SetTaskStatus(job.ID, user.ID, models.TaskStatusInProgress)
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatusInProgress, job.Status)
}

func TestBacklogJob_DoneIsInvalid(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc, space, user := newDBTaskService(tx)

	job, err := svc.CreateTask(space.ID, user.ID, &dtos.CreateTaskRequest{
		Title:  "Backlog Job",
		Type:   models.TaskTypeJob,
		Status: backlogStatusPtr(),
	})
	require.NoError(t, err)

	_, err = svc.SetTaskStatus(job.ID, user.ID, models.TaskStatusDone)
	require.Error(t, err, "backlog -> done must be rejected")
	assert.Contains(t, err.Error(), "backlog")

	// Status must be unchanged in the database.
	var got models.Task
	require.NoError(t, tx.First(&got, "id = ?", job.ID).Error)
	assert.Equal(t, models.TaskStatusBacklog, got.Status)
}

func TestBacklogTransitions_RevertRules(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc, space, user := newDBTaskService(tx)

	job, err := svc.CreateTask(space.ID, user.ID, &dtos.CreateTaskRequest{
		Title: "Open Job",
		Type:  models.TaskTypeJob,
	})
	require.NoError(t, err)
	require.Equal(t, models.TaskStatusOpen, job.Status)

	// open -> backlog (unschedule) is allowed.
	job, err = svc.SetTaskStatus(job.ID, user.ID, models.TaskStatusBacklog)
	require.NoError(t, err)
	assert.Equal(t, models.TaskStatusBacklog, job.Status)

	// backlog -> in_progress, then in_progress -> backlog is NOT allowed.
	job, err = svc.SetTaskStatus(job.ID, user.ID, models.TaskStatusInProgress)
	require.NoError(t, err)
	_, err = svc.SetTaskStatus(job.ID, user.ID, models.TaskStatusBacklog)
	require.Error(t, err, "in_progress -> backlog must be rejected")
}

func TestListJobs_FilterByBacklogStatus(t *testing.T) {
	tx := dbtest.TestTx(t)
	svc, space, user := newDBTaskService(tx)

	backlogJob, err := svc.CreateTask(space.ID, user.ID, &dtos.CreateTaskRequest{
		Title:  "Backlog Job",
		Type:   models.TaskTypeJob,
		Status: backlogStatusPtr(),
	})
	require.NoError(t, err)

	_, err = svc.CreateTask(space.ID, user.ID, &dtos.CreateTaskRequest{
		Title: "Open Job",
		Type:  models.TaskTypeJob,
	})
	require.NoError(t, err)

	jobType := models.TaskTypeJob
	backlog := models.TaskStatusBacklog
	tasks, total, err := svc.ListTasks(repositories.TaskFilter{
		SpaceID:  &space.ID,
		Type:     &jobType,
		Status:   &backlog,
		RootOnly: true,
		Limit:    50,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, tasks, 1)
	assert.Equal(t, backlogJob.ID, tasks[0].ID)
	assert.Equal(t, models.TaskStatusBacklog, tasks[0].Status)

	// Filtering by open must not return the backlog job.
	open := models.TaskStatusOpen
	tasks, _, err = svc.ListTasks(repositories.TaskFilter{
		SpaceID:  &space.ID,
		Type:     &jobType,
		Status:   &open,
		RootOnly: true,
		Limit:    50,
	})
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, "Open Job", tasks[0].Title)
}

// Ensure GetReadyTasks never surfaces backlog tasks (backlog is unscheduled).
func TestReadyWork_ExcludesBacklog(t *testing.T) {
	tx := dbtest.TestTx(t)
	space := dbfactory.Space(tx)
	user := dbfactory.User(tx)

	dbfactory.Task(tx, func(m *models.Task) {
		m.SpaceID = space.ID
		m.CreatedByID = user.ID
		m.Status = models.TaskStatusBacklog
		m.Title = "Backlog Task"
	})
	openTask := dbfactory.Task(tx, func(m *models.Task) {
		m.SpaceID = space.ID
		m.CreatedByID = user.ID
		m.Status = models.TaskStatusOpen
		m.Title = "Open Task"
	})

	readySvc := NewReadyWorkService(tx)
	ready, err := readySvc.GetReadyTasks(space.ID, nil)
	require.NoError(t, err)
	require.Len(t, ready, 1)
	assert.Equal(t, openTask.ID, ready[0].ID)
}
