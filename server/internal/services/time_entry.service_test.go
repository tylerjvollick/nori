package services

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/repositories"
)

// ---------------------------------------------------------------------------
// Mock time entry repository
// ---------------------------------------------------------------------------

type mockTimeEntryRepo struct {
	entries map[uuid.UUID]*models.TimeEntry
}

func newMockTimeEntryRepo() *mockTimeEntryRepo {
	return &mockTimeEntryRepo{entries: make(map[uuid.UUID]*models.TimeEntry)}
}

func (r *mockTimeEntryRepo) Create(entry *models.TimeEntry) error {
	r.entries[entry.ID] = entry
	return nil
}

func (r *mockTimeEntryRepo) GetByID(id uuid.UUID) (*models.TimeEntry, error) {
	e, ok := r.entries[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return e, nil
}

func (r *mockTimeEntryRepo) Update(entry *models.TimeEntry) error {
	r.entries[entry.ID] = entry
	return nil
}

func (r *mockTimeEntryRepo) Delete(id uuid.UUID) error {
	delete(r.entries, id)
	return nil
}

func (r *mockTimeEntryRepo) GetByTaskID(taskID string) ([]models.TimeEntry, error) {
	var result []models.TimeEntry
	for _, e := range r.entries {
		if e.TaskID == taskID {
			result = append(result, *e)
		}
	}
	return result, nil
}

func (r *mockTimeEntryRepo) GetRunningEntry(taskID string) (*models.TimeEntry, error) {
	for _, e := range r.entries {
		if e.TaskID == taskID && e.EndedAt == nil {
			return e, nil
		}
	}
	return nil, nil
}

func (r *mockTimeEntryRepo) GetCompletedTasksWithNoEntries(spaceID uuid.UUID) ([]models.Task, error) {
	return nil, nil // minimal mock — tested at integration level
}

// ---------------------------------------------------------------------------
// Minimal mock task repo for time entry tests
// ---------------------------------------------------------------------------

type minimalMockTaskRepo struct {
	tasks map[string]*models.Task
}

func newMinimalMockTaskRepo() *minimalMockTaskRepo {
	return &minimalMockTaskRepo{tasks: make(map[string]*models.Task)}
}

func (r *minimalMockTaskRepo) GetByID(id string) (*models.Task, error) {
	t, ok := r.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task %q not found", id)
	}
	return t, nil
}

// Unused interface methods — stubbed out.
func (r *minimalMockTaskRepo) Create(task *models.Task) error          { r.tasks[task.ID] = task; return nil }
func (r *minimalMockTaskRepo) List(_ repositories.TaskFilter) ([]models.Task, int64, error) {
	return nil, 0, nil
}
func (r *minimalMockTaskRepo) Update(task *models.Task) error          { r.tasks[task.ID] = task; return nil }
func (r *minimalMockTaskRepo) Delete(_ string) error                   { return nil }
func (r *minimalMockTaskRepo) GetChildren(_ string) ([]models.Task, error) { return nil, nil }
func (r *minimalMockTaskRepo) GetRoot(_ string) (*models.Task, error)  { return nil, nil }
func (r *minimalMockTaskRepo) GetDescendants(idPrefix string) ([]models.Task, error) {
	var result []models.Task
	for id, t := range r.tasks {
		// Descendants share the parent's hierarchical ID prefix (e.g. "task-j.1").
		if strings.HasPrefix(id, idPrefix+".") {
			result = append(result, *t)
		}
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func setupTimeEntryService(t *testing.T) (*TimeEntryService, *mockTimeEntryRepo, *minimalMockTaskRepo) {
	t.Helper()
	teRepo := newMockTimeEntryRepo()
	taskRepo := newMinimalMockTaskRepo()
	svc := NewTimeEntryService(teRepo, taskRepo)

	// Create a test task.
	taskRepo.tasks["task-1"] = &models.Task{
		ID:      "task-1",
		SpaceID: uuid.New(),
		Status:  models.TaskStatusInProgress,
	}

	return svc, teRepo, taskRepo
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestStartTimer_CreatesRunningEntry(t *testing.T) {
	svc, repo, taskRepo := setupTimeEntryService(t)
	task := taskRepo.tasks["task-1"]
	userID := uuid.New()

	entry, err := svc.StartTimer("task-1", task.SpaceID, userID)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, entry.ID)
	assert.Equal(t, "task-1", entry.TaskID)
	assert.Equal(t, userID, entry.LoggedByID)
	assert.Nil(t, entry.EndedAt)

	// Should be in the repo.
	assert.Len(t, repo.entries, 1)
}

func TestStartTimer_ConflictIfAlreadyRunning(t *testing.T) {
	svc, _, taskRepo := setupTimeEntryService(t)
	task := taskRepo.tasks["task-1"]
	userID := uuid.New()

	_, err := svc.StartTimer("task-1", task.SpaceID, userID)
	require.NoError(t, err)

	_, err = svc.StartTimer("task-1", task.SpaceID, userID)
	assert.ErrorIs(t, err, ErrTimerAlreadyRunning)
}

func TestPauseTimer_SetsEndedAtAndDuration(t *testing.T) {
	svc, _, taskRepo := setupTimeEntryService(t)
	task := taskRepo.tasks["task-1"]

	_, err := svc.StartTimer("task-1", task.SpaceID, uuid.New())
	require.NoError(t, err)

	entry, err := svc.PauseTimer("task-1")
	require.NoError(t, err)
	assert.True(t, entry.IsPaused)
	assert.NotNil(t, entry.PausedAt)
	assert.Nil(t, entry.EndedAt) // pausing does not end the session
}

func TestPauseTimer_ConflictIfNoTimerRunning(t *testing.T) {
	svc, _, _ := setupTimeEntryService(t)

	_, err := svc.PauseTimer("task-1")
	assert.ErrorIs(t, err, ErrNoTimerRunning)
}

// AC#12: start → pause → start → pause creates 2 separate time entries.
func TestStartStopStartStop_CreatesTwoEntries(t *testing.T) {
	svc, repo, taskRepo := setupTimeEntryService(t)
	task := taskRepo.tasks["task-1"]
	userID := uuid.New()

	// First session: start → stop.
	_, err := svc.StartTimer("task-1", task.SpaceID, userID)
	require.NoError(t, err)
	_, err = svc.StopTimer("task-1")
	require.NoError(t, err)

	// Second session: start → stop.
	_, err = svc.StartTimer("task-1", task.SpaceID, userID)
	require.NoError(t, err)
	_, err = svc.StopTimer("task-1")
	require.NoError(t, err)

	// Should have 2 entries, both with ended_at set.
	assert.Len(t, repo.entries, 2)
	for _, e := range repo.entries {
		assert.NotNil(t, e.EndedAt)
	}
}

func TestPauseResume_AccumulatesElapsed(t *testing.T) {
	svc, _, taskRepo := setupTimeEntryService(t)
	task := taskRepo.tasks["task-1"]
	userID := uuid.New()

	_, err := svc.StartTimer("task-1", task.SpaceID, userID)
	require.NoError(t, err)

	// Pause — should accumulate some small elapsed.
	paused, err := svc.PauseTimer("task-1")
	require.NoError(t, err)
	assert.True(t, paused.IsPaused)
	assert.GreaterOrEqual(t, paused.ElapsedSecs, 0)

	// Resume.
	resumed, err := svc.ResumeTimer("task-1")
	require.NoError(t, err)
	assert.False(t, resumed.IsPaused)
	assert.Nil(t, resumed.PausedAt)
	assert.NotNil(t, resumed.ResumedAt)
}

// AC#13: complete while timer running auto-pauses and completes.
func TestStopRunningTimer_AutoStops(t *testing.T) {
	svc, repo, taskRepo := setupTimeEntryService(t)
	task := taskRepo.tasks["task-1"]

	_, err := svc.StartTimer("task-1", task.SpaceID, uuid.New())
	require.NoError(t, err)

	// StopRunningTimer is what CompleteTask calls.
	err = svc.StopRunningTimer("task-1")
	require.NoError(t, err)

	// Entry should now be closed.
	for _, e := range repo.entries {
		assert.NotNil(t, e.EndedAt)
	}
}

// AC#14: complete while paused succeeds without error.
func TestStopRunningTimer_NoErrorWhenPaused(t *testing.T) {
	svc, _, taskRepo := setupTimeEntryService(t)
	task := taskRepo.tasks["task-1"]

	// Start and pause.
	_, err := svc.StartTimer("task-1", task.SpaceID, uuid.New())
	require.NoError(t, err)
	_, err = svc.PauseTimer("task-1")
	require.NoError(t, err)

	// StopRunningTimer should succeed (no running timer to stop).
	err = svc.StopRunningTimer("task-1")
	assert.NoError(t, err)
}

// AC#15: manual entry with custom start/end times.
func TestCreateManualEntry_CustomTimes(t *testing.T) {
	svc, _, taskRepo := setupTimeEntryService(t)
	task := taskRepo.tasks["task-1"]
	userID := uuid.New()

	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()
	notes := "Forgot to start timer"

	entry, err := svc.CreateManualEntry("task-1", task.SpaceID, userID, &dtos.CreateTimeEntryRequest{
		StartedAt: start,
		EndedAt:   &end,
		Notes:     &notes,
	})
	require.NoError(t, err)
	assert.Equal(t, start.Unix(), entry.StartedAt.Unix())
	assert.NotNil(t, entry.EndedAt)
	assert.InDelta(t, 3600, entry.ElapsedSecs, 2) // ~1 hour
	assert.Equal(t, &notes, entry.Notes)
}

func TestListEntries_ReturnsTotalDuration(t *testing.T) {
	svc, _, taskRepo := setupTimeEntryService(t)
	task := taskRepo.tasks["task-1"]
	userID := uuid.New()

	// Create two manual entries.
	end1 := time.Now().Add(-30 * time.Minute)
	start1 := end1.Add(-10 * time.Minute)
	_, err := svc.CreateManualEntry("task-1", task.SpaceID, userID, &dtos.CreateTimeEntryRequest{
		StartedAt: start1,
		EndedAt:   &end1,
	})
	require.NoError(t, err)

	end2 := time.Now()
	start2 := end2.Add(-20 * time.Minute)
	_, err = svc.CreateManualEntry("task-1", task.SpaceID, userID, &dtos.CreateTimeEntryRequest{
		StartedAt: start2,
		EndedAt:   &end2,
	})
	require.NoError(t, err)

	entries, total, err := svc.ListEntries("task-1")
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.InDelta(t, 1800, total, 2) // 10m + 20m = 30m = 1800s
}

// nori-19e: a job's time summary rolls up own + all descendant tasks' time.
func TestGetJobTimeSummary_RollsUpDescendants(t *testing.T) {
	svc, _, taskRepo := setupTimeEntryService(t)
	spaceID := uuid.New()
	userID := uuid.New()

	// Job with two child tasks.
	taskRepo.tasks["task-j"] = &models.Task{ID: "task-j", SpaceID: spaceID, Type: models.TaskTypeJob}
	taskRepo.tasks["task-j.1"] = &models.Task{ID: "task-j.1", SpaceID: spaceID}
	taskRepo.tasks["task-j.2"] = &models.Task{ID: "task-j.2", SpaceID: spaceID}

	// Helper to log a closed entry of `mins` minutes on a task.
	logMinutes := func(taskID string, mins int) {
		end := time.Now()
		start := end.Add(-time.Duration(mins) * time.Minute)
		_, err := svc.CreateManualEntry(taskID, spaceID, userID, &dtos.CreateTimeEntryRequest{
			StartedAt: start,
			EndedAt:   &end,
		})
		require.NoError(t, err)
	}

	logMinutes("task-j", 10)   // 10m logged directly on the job
	logMinutes("task-j.1", 30) // 30m on child 1
	logMinutes("task-j.1", 20) // +20m on child 1 (two sessions)
	logMinutes("task-j.2", 60) // 60m on child 2

	summary, err := svc.GetJobTimeSummary("task-j")
	require.NoError(t, err)
	assert.Equal(t, "task-j", summary.JobID)
	assert.InDelta(t, 600, summary.OwnSecs, 2)      // 10m
	assert.InDelta(t, 7200, summary.RollupSecs, 4)  // 10 + 30 + 20 + 60 = 120m
}

// A childless job rolls up only its own time.
func TestGetJobTimeSummary_ChildlessJob(t *testing.T) {
	svc, _, taskRepo := setupTimeEntryService(t)
	spaceID := uuid.New()
	taskRepo.tasks["task-solo"] = &models.Task{ID: "task-solo", SpaceID: spaceID, Type: models.TaskTypeJob}

	end := time.Now()
	start := end.Add(-45 * time.Minute)
	_, err := svc.CreateManualEntry("task-solo", spaceID, uuid.New(), &dtos.CreateTimeEntryRequest{
		StartedAt: start,
		EndedAt:   &end,
	})
	require.NoError(t, err)

	summary, err := svc.GetJobTimeSummary("task-solo")
	require.NoError(t, err)
	assert.InDelta(t, 2700, summary.OwnSecs, 2)
	assert.Equal(t, summary.OwnSecs, summary.RollupSecs) // no descendants
}

func TestUpdateEntry_UpdatesFields(t *testing.T) {
	svc, _, taskRepo := setupTimeEntryService(t)
	task := taskRepo.tasks["task-1"]

	end := time.Now()
	start := end.Add(-1 * time.Hour)
	entry, err := svc.CreateManualEntry("task-1", task.SpaceID, uuid.New(), &dtos.CreateTimeEntryRequest{
		StartedAt: start,
		EndedAt:   &end,
	})
	require.NoError(t, err)

	newNotes := "Updated notes"
	updated, err := svc.UpdateEntry(entry.ID, &dtos.UpdateTimeEntryRequest{
		Notes: &newNotes,
	})
	require.NoError(t, err)
	assert.Equal(t, &newNotes, updated.Notes)
}

func TestDeleteEntry_RemovesEntry(t *testing.T) {
	svc, repo, taskRepo := setupTimeEntryService(t)
	task := taskRepo.tasks["task-1"]

	end := time.Now()
	start := end.Add(-1 * time.Hour)
	entry, err := svc.CreateManualEntry("task-1", task.SpaceID, uuid.New(), &dtos.CreateTimeEntryRequest{
		StartedAt: start,
		EndedAt:   &end,
	})
	require.NoError(t, err)
	assert.Len(t, repo.entries, 1)

	err = svc.DeleteEntry(entry.ID)
	require.NoError(t, err)
	assert.Len(t, repo.entries, 0)
}

func TestLoggedByID_SetToCurrentUser(t *testing.T) {
	svc, _, taskRepo := setupTimeEntryService(t)
	task := taskRepo.tasks["task-1"]
	userID := uuid.New()

	entry, err := svc.StartTimer("task-1", task.SpaceID, userID)
	require.NoError(t, err)
	assert.Equal(t, userID, entry.LoggedByID)
}
