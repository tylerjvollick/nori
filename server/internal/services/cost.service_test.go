package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/tylerjvollick/nori/internal/models"
)

// ── Mock repositories ──────────────────────────────────────────────────────

type mockCostEntryRepo struct {
	entries []models.CostEntry
}

func (m *mockCostEntryRepo) Create(entry *models.CostEntry) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	entry.CreatedAt = time.Now()
	m.entries = append(m.entries, *entry)
	return nil
}

func (m *mockCostEntryRepo) GetByID(id uuid.UUID) (*models.CostEntry, error) {
	for _, e := range m.entries {
		if e.ID == id {
			return &e, nil
		}
	}
	return nil, fmt.Errorf("cost entry not found")
}

func (m *mockCostEntryRepo) GetByTaskID(taskID string) ([]models.CostEntry, error) {
	var result []models.CostEntry
	for _, e := range m.entries {
		if e.TaskID == taskID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (m *mockCostEntryRepo) Delete(id uuid.UUID) error {
	for i, e := range m.entries {
		if e.ID == id {
			m.entries = append(m.entries[:i], m.entries[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockCostEntryRepo) DeleteByTaskIDAndCostType(taskID string, costType models.CostType) error {
	var remaining []models.CostEntry
	for _, e := range m.entries {
		if !(e.TaskID == taskID && e.CostType == costType) {
			remaining = append(remaining, e)
		}
	}
	m.entries = remaining
	return nil
}

type mockCostTimeEventRepo struct {
	events map[string][]models.TimeEvent // taskID → events
}

func (m *mockCostTimeEventRepo) GetByTaskID(taskID string) ([]models.TimeEvent, error) {
	return m.events[taskID], nil
}

type mockCostTaskRepo struct {
	tasks       map[string]*models.Task
	descendants map[string][]models.Task
}

func (m *mockCostTaskRepo) GetByID(id string) (*models.Task, error) {
	task, ok := m.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found")
	}
	return task, nil
}

func (m *mockCostTaskRepo) GetDescendants(idPrefix string) ([]models.Task, error) {
	return m.descendants[idPrefix], nil
}

type mockCostSpaceRepo struct {
	spaces map[uuid.UUID]*models.Space
}

func (m *mockCostSpaceRepo) FindByID(id uuid.UUID) (*models.Space, error) {
	space, ok := m.spaces[id]
	if !ok {
		return nil, fmt.Errorf("space not found")
	}
	return space, nil
}

type mockCostStationRepo struct {
	stations map[uuid.UUID][]models.Station // spaceID → stations
}

func (m *mockCostStationRepo) GetBySpaceID(spaceID uuid.UUID) ([]models.Station, error) {
	return m.stations[spaceID], nil
}

// ── Helpers ────────────────────────────────────────────────────────────────

func newTestCostService() (*CostService, *mockCostEntryRepo, *mockCostTimeEventRepo, *mockCostTaskRepo, *mockCostSpaceRepo) {
	costRepo := &mockCostEntryRepo{}
	timeRepo := &mockCostTimeEventRepo{events: make(map[string][]models.TimeEvent)}
	taskRepo := &mockCostTaskRepo{tasks: make(map[string]*models.Task), descendants: make(map[string][]models.Task)}
	spaceRepo := &mockCostSpaceRepo{spaces: make(map[uuid.UUID]*models.Space)}
	stationRepo := &mockCostStationRepo{stations: make(map[uuid.UUID][]models.Station)}
	svc := NewCostService(costRepo, timeRepo, taskRepo, spaceRepo, stationRepo)
	return svc, costRepo, timeRepo, taskRepo, spaceRepo
}

func newTestCostServiceWithStations() (*CostService, *mockCostEntryRepo, *mockCostTimeEventRepo, *mockCostTaskRepo, *mockCostSpaceRepo, *mockCostStationRepo) {
	costRepo := &mockCostEntryRepo{}
	timeRepo := &mockCostTimeEventRepo{events: make(map[string][]models.TimeEvent)}
	taskRepo := &mockCostTaskRepo{tasks: make(map[string]*models.Task), descendants: make(map[string][]models.Task)}
	spaceRepo := &mockCostSpaceRepo{spaces: make(map[uuid.UUID]*models.Space)}
	stationRepo := &mockCostStationRepo{stations: make(map[uuid.UUID][]models.Station)}
	svc := NewCostService(costRepo, timeRepo, taskRepo, spaceRepo, stationRepo)
	return svc, costRepo, timeRepo, taskRepo, spaceRepo, stationRepo
}

func makeRate(rate string) *decimal.Decimal {
	d, _ := decimal.NewFromString(rate)
	return &d
}

func makeTimeEvent(taskID string, eventType models.TimeEventType, ts time.Time) models.TimeEvent {
	return models.TimeEvent{
		ID:        uuid.New(),
		SpaceID:   uuid.New(),
		UserID:    uuid.New(),
		TaskID:    &taskID,
		EventType: eventType,
		Source:    models.TimeEventSourceSystem,
		Timestamp: ts,
	}
}

// ── Tests ──────────────────────────────────────────────────────────────────

func TestComputeActiveSeconds_SimpleCheckInOut(t *testing.T) {
	base := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	events := []models.TimeEvent{
		makeTimeEvent("t1", models.TimeEventTypeCheckIn, base),
		makeTimeEvent("t1", models.TimeEventTypeCheckOut, base.Add(1*time.Hour)),
	}
	got := computeActiveSeconds(events)
	if got != 3600 {
		t.Errorf("expected 3600 active seconds, got %d", got)
	}
}

func TestComputeActiveSeconds_WithPauseResume(t *testing.T) {
	base := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	events := []models.TimeEvent{
		makeTimeEvent("t1", models.TimeEventTypeCheckIn, base),
		makeTimeEvent("t1", models.TimeEventTypePause, base.Add(30*time.Minute)),
		makeTimeEvent("t1", models.TimeEventTypeResume, base.Add(45*time.Minute)),
		makeTimeEvent("t1", models.TimeEventTypeCheckOut, base.Add(1*time.Hour)),
	}
	got := computeActiveSeconds(events)
	// 30 min active + 15 min active = 45 min = 2700 secs
	if got != 2700 {
		t.Errorf("expected 2700 active seconds, got %d", got)
	}
}

func TestComputeActiveSeconds_NoEvents(t *testing.T) {
	got := computeActiveSeconds(nil)
	if got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestComputeActiveSeconds_UnsortedEvents(t *testing.T) {
	base := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	// Events in reverse order — should still compute correctly.
	events := []models.TimeEvent{
		makeTimeEvent("t1", models.TimeEventTypeCheckOut, base.Add(1*time.Hour)),
		makeTimeEvent("t1", models.TimeEventTypeCheckIn, base),
	}
	got := computeActiveSeconds(events)
	if got != 3600 {
		t.Errorf("expected 3600, got %d", got)
	}
}

func TestComputeTaskLaborCost_Basic(t *testing.T) {
	svc, _, timeRepo, taskRepo, spaceRepo := newTestCostService()

	spaceID := uuid.New()
	userID := uuid.New()
	taskID := "job-1.1"

	taskRepo.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}
	spaceRepo.spaces[spaceID] = &models.Space{ID: spaceID, DefaultLaborRate: makeRate("50.00")}

	base := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	timeRepo.events[taskID] = []models.TimeEvent{
		makeTimeEvent(taskID, models.TimeEventTypeCheckIn, base),
		makeTimeEvent(taskID, models.TimeEventTypeCheckOut, base.Add(2*time.Hour)),
	}

	result, err := svc.ComputeTaskLaborCost(taskID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ActiveSecs != 7200 {
		t.Errorf("expected 7200 active secs, got %d", result.ActiveSecs)
	}

	expectedCost := decimal.NewFromFloat(100.00)
	if !result.LaborCost.Equal(expectedCost) {
		t.Errorf("expected labor cost %s, got %s", expectedCost, result.LaborCost)
	}

	if result.CostEntryID == uuid.Nil {
		t.Error("expected non-nil cost entry ID")
	}
}

func TestComputeTaskLaborCost_NoLaborRate(t *testing.T) {
	svc, _, _, taskRepo, spaceRepo := newTestCostService()

	spaceID := uuid.New()
	taskID := "job-1.1"
	taskRepo.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}
	spaceRepo.spaces[spaceID] = &models.Space{ID: spaceID, DefaultLaborRate: nil}

	_, err := svc.ComputeTaskLaborCost(taskID, uuid.New())
	if err == nil {
		t.Fatal("expected error for missing labor rate")
	}
}

func TestComputeTaskLaborCost_ReplacesExisting(t *testing.T) {
	svc, costRepo, timeRepo, taskRepo, spaceRepo := newTestCostService()

	spaceID := uuid.New()
	userID := uuid.New()
	taskID := "job-1.1"

	taskRepo.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}
	spaceRepo.spaces[spaceID] = &models.Space{ID: spaceID, DefaultLaborRate: makeRate("50.00")}

	base := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	timeRepo.events[taskID] = []models.TimeEvent{
		makeTimeEvent(taskID, models.TimeEventTypeCheckIn, base),
		makeTimeEvent(taskID, models.TimeEventTypeCheckOut, base.Add(1*time.Hour)),
	}

	// First computation.
	_, err := svc.ComputeTaskLaborCost(taskID, userID)
	if err != nil {
		t.Fatalf("first compute: %v", err)
	}

	// Second computation should replace, not duplicate.
	_, err = svc.ComputeTaskLaborCost(taskID, userID)
	if err != nil {
		t.Fatalf("second compute: %v", err)
	}

	entries, _ := costRepo.GetByTaskID(taskID)
	laborEntries := 0
	for _, e := range entries {
		if e.CostType == models.CostTypeLabor {
			laborEntries++
		}
	}
	if laborEntries != 1 {
		t.Errorf("expected 1 labor entry after recompute, got %d", laborEntries)
	}
}

func TestComputeJobLaborCost_AggregatesChildren(t *testing.T) {
	svc, _, timeRepo, taskRepo, spaceRepo := newTestCostService()

	spaceID := uuid.New()
	userID := uuid.New()
	jobID := "job-1"
	childID1 := "job-1.1"
	childID2 := "job-1.2"

	taskRepo.tasks[jobID] = &models.Task{ID: jobID, SpaceID: spaceID, Type: models.TaskTypeJob}
	taskRepo.tasks[childID1] = &models.Task{ID: childID1, SpaceID: spaceID, ParentID: &jobID}
	taskRepo.tasks[childID2] = &models.Task{ID: childID2, SpaceID: spaceID, ParentID: &jobID}
	taskRepo.descendants[jobID] = []models.Task{*taskRepo.tasks[childID1], *taskRepo.tasks[childID2]}
	spaceRepo.spaces[spaceID] = &models.Space{ID: spaceID, DefaultLaborRate: makeRate("60.00")}

	base := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	// Child 1: 1 hour
	timeRepo.events[childID1] = []models.TimeEvent{
		makeTimeEvent(childID1, models.TimeEventTypeCheckIn, base),
		makeTimeEvent(childID1, models.TimeEventTypeCheckOut, base.Add(1*time.Hour)),
	}
	// Child 2: 30 minutes
	timeRepo.events[childID2] = []models.TimeEvent{
		makeTimeEvent(childID2, models.TimeEventTypeCheckIn, base),
		makeTimeEvent(childID2, models.TimeEventTypeCheckOut, base.Add(30*time.Minute)),
	}
	// Job root: no direct time events.

	result, err := svc.ComputeJobLaborCost(jobID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.TaskResults) != 2 {
		t.Errorf("expected 2 task results, got %d", len(result.TaskResults))
	}

	// 1h × $60 + 0.5h × $60 = $60 + $30 = $90
	expectedTotal := decimal.NewFromFloat(90.00)
	if !result.TotalCost.Equal(expectedTotal) {
		t.Errorf("expected total cost %s, got %s", expectedTotal, result.TotalCost)
	}

	expectedSecs := int64(5400) // 1.5 hours
	if result.TotalSecs != expectedSecs {
		t.Errorf("expected %d total secs, got %d", expectedSecs, result.TotalSecs)
	}
}

func TestComputeJobLaborCost_SkipsTasksWithNoEvents(t *testing.T) {
	svc, _, _, taskRepo, spaceRepo := newTestCostService()

	spaceID := uuid.New()
	jobID := "job-1"
	childID := "job-1.1"

	taskRepo.tasks[jobID] = &models.Task{ID: jobID, SpaceID: spaceID}
	taskRepo.tasks[childID] = &models.Task{ID: childID, SpaceID: spaceID, ParentID: &jobID}
	taskRepo.descendants[jobID] = []models.Task{*taskRepo.tasks[childID]}
	spaceRepo.spaces[spaceID] = &models.Space{ID: spaceID, DefaultLaborRate: makeRate("50.00")}
	// No time events for any task.

	result, err := svc.ComputeJobLaborCost(jobID, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.TaskResults) != 0 {
		t.Errorf("expected 0 task results, got %d", len(result.TaskResults))
	}

	if !result.TotalCost.Equal(decimal.Zero) {
		t.Errorf("expected zero total cost, got %s", result.TotalCost)
	}
}

func TestEstimateRecipeLaborCost_Basic(t *testing.T) {
	svc, _, _, taskRepo, spaceRepo := newTestCostService()

	spaceID := uuid.New()
	rootID := "recipe-root"

	est1 := 3600 // 1 hour
	est2 := 1800 // 30 min

	taskRepo.tasks[rootID] = &models.Task{ID: rootID, SpaceID: spaceID, EstimatedTimeSecs: &est1}
	child := models.Task{ID: rootID + ".1", SpaceID: spaceID, ParentID: &rootID, EstimatedTimeSecs: &est2}
	taskRepo.tasks[child.ID] = &child
	taskRepo.descendants[rootID] = []models.Task{child}
	spaceRepo.spaces[spaceID] = &models.Space{ID: spaceID, DefaultLaborRate: makeRate("40.00")}

	result, err := svc.EstimateRecipeLaborCost(rootID, spaceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedSecs := int64(5400) // 1.5 hours
	if result.TotalEstimatedSecs != expectedSecs {
		t.Errorf("expected %d secs, got %d", expectedSecs, result.TotalEstimatedSecs)
	}

	// 1.5h × $40 = $60
	expectedCost := decimal.NewFromFloat(60.00)
	if !result.EstimatedCost.Equal(expectedCost) {
		t.Errorf("expected cost %s, got %s", expectedCost, result.EstimatedCost)
	}
}

func TestEstimateRecipeLaborCost_NoEstimates(t *testing.T) {
	svc, _, _, taskRepo, spaceRepo := newTestCostService()

	spaceID := uuid.New()
	rootID := "recipe-root"

	taskRepo.tasks[rootID] = &models.Task{ID: rootID, SpaceID: spaceID}
	taskRepo.descendants[rootID] = nil
	spaceRepo.spaces[spaceID] = &models.Space{ID: spaceID, DefaultLaborRate: makeRate("40.00")}

	result, err := svc.EstimateRecipeLaborCost(rootID, spaceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalEstimatedSecs != 0 {
		t.Errorf("expected 0 secs, got %d", result.TotalEstimatedSecs)
	}

	if !result.EstimatedCost.Equal(decimal.Zero) {
		t.Errorf("expected zero cost, got %s", result.EstimatedCost)
	}
}

func TestEstimateRecipeLaborCost_NoLaborRate(t *testing.T) {
	svc, _, _, taskRepo, spaceRepo := newTestCostService()

	spaceID := uuid.New()
	rootID := "recipe-root"

	taskRepo.tasks[rootID] = &models.Task{ID: rootID, SpaceID: spaceID}
	spaceRepo.spaces[spaceID] = &models.Space{ID: spaceID, DefaultLaborRate: nil}

	_, err := svc.EstimateRecipeLaborCost(rootID, spaceID)
	if err == nil {
		t.Fatal("expected error for missing labor rate")
	}
}

func TestGetTaskCosts(t *testing.T) {
	svc, costRepo, _, _, _ := newTestCostService()

	taskID := "job-1.1"
	costRepo.entries = []models.CostEntry{
		{ID: uuid.New(), TaskID: taskID, CostType: models.CostTypeLabor, Amount: decimal.NewFromFloat(50.00)},
		{ID: uuid.New(), TaskID: taskID, CostType: models.CostTypeMaterial, Amount: decimal.NewFromFloat(25.00)},
		{ID: uuid.New(), TaskID: "other-task", CostType: models.CostTypeLabor, Amount: decimal.NewFromFloat(10.00)},
	}

	entries, err := svc.GetTaskCosts(taskID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestComputeActiveSeconds_MultiplePauseCycles(t *testing.T) {
	base := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	events := []models.TimeEvent{
		makeTimeEvent("t1", models.TimeEventTypeCheckIn, base),
		makeTimeEvent("t1", models.TimeEventTypePause, base.Add(10*time.Minute)),
		makeTimeEvent("t1", models.TimeEventTypeResume, base.Add(20*time.Minute)),
		makeTimeEvent("t1", models.TimeEventTypePause, base.Add(30*time.Minute)),
		makeTimeEvent("t1", models.TimeEventTypeResume, base.Add(40*time.Minute)),
		makeTimeEvent("t1", models.TimeEventTypeCheckOut, base.Add(50*time.Minute)),
	}
	// Active: 0-10, 20-30, 40-50 = 30 min = 1800 secs
	got := computeActiveSeconds(events)
	if got != 1800 {
		t.Errorf("expected 1800, got %d", got)
	}
}

func TestComputeTaskLaborCost_WithPauseResume(t *testing.T) {
	svc, _, timeRepo, taskRepo, spaceRepo := newTestCostService()

	spaceID := uuid.New()
	taskID := "job-1.1"

	taskRepo.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}
	spaceRepo.spaces[spaceID] = &models.Space{ID: spaceID, DefaultLaborRate: makeRate("120.00")}

	base := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	timeRepo.events[taskID] = []models.TimeEvent{
		makeTimeEvent(taskID, models.TimeEventTypeCheckIn, base),
		makeTimeEvent(taskID, models.TimeEventTypePause, base.Add(30*time.Minute)),
		makeTimeEvent(taskID, models.TimeEventTypeResume, base.Add(1*time.Hour)),
		makeTimeEvent(taskID, models.TimeEventTypeCheckOut, base.Add(90*time.Minute)),
	}

	result, err := svc.ComputeTaskLaborCost(taskID, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Active: 30 min + 30 min = 1 hour = 3600 secs
	if result.ActiveSecs != 3600 {
		t.Errorf("expected 3600 active secs, got %d", result.ActiveSecs)
	}

	// 1h × $120 = $120
	expectedCost := decimal.NewFromFloat(120.00)
	if !result.LaborCost.Equal(expectedCost) {
		t.Errorf("expected labor cost %s, got %s", expectedCost, result.LaborCost)
	}
}

// ── GetJobCostSummary Tests ──────────────────────────────────────────────

func TestGetJobCostSummary_Basic(t *testing.T) {
	svc, _, timeRepo, taskRepo, spaceRepo, stationRepo := newTestCostServiceWithStations()

	spaceID := uuid.New()
	stationID := uuid.New()
	jobID := "job-1"
	childID := "job-1.1"

	est := 3600 // 1 hour estimated
	taskRepo.tasks[jobID] = &models.Task{ID: jobID, SpaceID: spaceID, Type: models.TaskTypeJob}
	taskRepo.tasks[childID] = &models.Task{ID: childID, SpaceID: spaceID, ParentID: &jobID, StationID: &stationID, EstimatedTimeSecs: &est}
	taskRepo.descendants[jobID] = []models.Task{*taskRepo.tasks[childID]}
	spaceRepo.spaces[spaceID] = &models.Space{ID: spaceID, DefaultLaborRate: makeRate("60.00")}
	stationRepo.stations[spaceID] = []models.Station{{ID: stationID, Name: "Assembly"}}

	base := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	timeRepo.events[childID] = []models.TimeEvent{
		makeTimeEvent(childID, models.TimeEventTypeCheckIn, base),
		makeTimeEvent(childID, models.TimeEventTypeCheckOut, base.Add(2*time.Hour)),
	}

	result, err := svc.GetJobCostSummary(jobID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.JobID != jobID {
		t.Errorf("expected jobId %q, got %q", jobID, result.JobID)
	}

	// Estimated: 1h × $60 = $60
	expectedEstHours := decimal.NewFromInt(1)
	if !result.Estimated.LaborHours.Equal(expectedEstHours) {
		t.Errorf("expected estimated hours %s, got %s", expectedEstHours, result.Estimated.LaborHours)
	}
	expectedEstCost := decimal.NewFromFloat(60.00)
	if !result.Estimated.LaborCost.Equal(expectedEstCost) {
		t.Errorf("expected estimated cost %s, got %s", expectedEstCost, result.Estimated.LaborCost)
	}

	// Actual: 2h × $60 = $120
	expectedActHours := decimal.NewFromInt(2)
	if !result.Actual.LaborHours.Equal(expectedActHours) {
		t.Errorf("expected actual hours %s, got %s", expectedActHours, result.Actual.LaborHours)
	}
	expectedActCost := decimal.NewFromFloat(120.00)
	if !result.Actual.LaborCost.Equal(expectedActCost) {
		t.Errorf("expected actual cost %s, got %s", expectedActCost, result.Actual.LaborCost)
	}

	// Variance: actual - estimated = +1h, +$60, +100%
	expectedVarHours := decimal.NewFromInt(1)
	if !result.Variance.LaborHours.Equal(expectedVarHours) {
		t.Errorf("expected variance hours %s, got %s", expectedVarHours, result.Variance.LaborHours)
	}
	expectedVarCost := decimal.NewFromFloat(60.00)
	if !result.Variance.LaborCost.Equal(expectedVarCost) {
		t.Errorf("expected variance cost %s, got %s", expectedVarCost, result.Variance.LaborCost)
	}
	expectedVarPct := decimal.NewFromInt(100)
	if !result.Variance.Percent.Equal(expectedVarPct) {
		t.Errorf("expected variance percent %s, got %s", expectedVarPct, result.Variance.Percent)
	}
}

func TestGetJobCostSummary_ByStation(t *testing.T) {
	svc, _, timeRepo, taskRepo, spaceRepo, stationRepo := newTestCostServiceWithStations()

	spaceID := uuid.New()
	st1 := uuid.New()
	st2 := uuid.New()
	jobID := "job-1"

	est1 := 3600 // 1 hour
	est2 := 1800 // 30 min
	child1 := "job-1.1"
	child2 := "job-1.2"

	taskRepo.tasks[jobID] = &models.Task{ID: jobID, SpaceID: spaceID, Type: models.TaskTypeJob}
	taskRepo.tasks[child1] = &models.Task{ID: child1, SpaceID: spaceID, ParentID: &jobID, StationID: &st1, EstimatedTimeSecs: &est1}
	taskRepo.tasks[child2] = &models.Task{ID: child2, SpaceID: spaceID, ParentID: &jobID, StationID: &st2, EstimatedTimeSecs: &est2}
	taskRepo.descendants[jobID] = []models.Task{*taskRepo.tasks[child1], *taskRepo.tasks[child2]}
	spaceRepo.spaces[spaceID] = &models.Space{ID: spaceID, DefaultLaborRate: makeRate("50.00")}
	stationRepo.stations[spaceID] = []models.Station{
		{ID: st1, Name: "Mill"},
		{ID: st2, Name: "Finish"},
	}

	base := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	timeRepo.events[child1] = []models.TimeEvent{
		makeTimeEvent(child1, models.TimeEventTypeCheckIn, base),
		makeTimeEvent(child1, models.TimeEventTypeCheckOut, base.Add(90*time.Minute)),
	}
	timeRepo.events[child2] = []models.TimeEvent{
		makeTimeEvent(child2, models.TimeEventTypeCheckIn, base),
		makeTimeEvent(child2, models.TimeEventTypeCheckOut, base.Add(20*time.Minute)),
	}

	result, err := svc.GetJobCostSummary(jobID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.ByStation) != 2 {
		t.Fatalf("expected 2 station summaries, got %d", len(result.ByStation))
	}

	// Find each station in the results (map iteration order is non-deterministic).
	stationMap := make(map[string]StationCostSummary)
	for _, s := range result.ByStation {
		stationMap[s.StationName] = s
	}

	mill := stationMap["Mill"]
	if !mill.EstimatedHours.Equal(decimal.NewFromInt(1)) {
		t.Errorf("Mill: expected 1h estimated, got %s", mill.EstimatedHours)
	}
	// 90 min = 1.5h
	expectedMillActual := decimal.NewFromFloat(1.5)
	if !mill.ActualHours.Equal(expectedMillActual) {
		t.Errorf("Mill: expected 1.5h actual, got %s", mill.ActualHours)
	}
	// variance: 1.5 - 1 = 0.5
	expectedMillVar := decimal.NewFromFloat(0.5)
	if !mill.Variance.Equal(expectedMillVar) {
		t.Errorf("Mill: expected 0.5h variance, got %s", mill.Variance)
	}

	finish := stationMap["Finish"]
	// 30 min estimated = 0.5h
	expectedFinishEst := decimal.NewFromFloat(0.5)
	if !finish.EstimatedHours.Equal(expectedFinishEst) {
		t.Errorf("Finish: expected 0.5h estimated, got %s", finish.EstimatedHours)
	}
}

func TestGetJobCostSummary_NoEstimates(t *testing.T) {
	svc, _, timeRepo, taskRepo, spaceRepo, stationRepo := newTestCostServiceWithStations()

	spaceID := uuid.New()
	jobID := "job-1"
	childID := "job-1.1"

	taskRepo.tasks[jobID] = &models.Task{ID: jobID, SpaceID: spaceID, Type: models.TaskTypeJob}
	taskRepo.tasks[childID] = &models.Task{ID: childID, SpaceID: spaceID, ParentID: &jobID}
	taskRepo.descendants[jobID] = []models.Task{*taskRepo.tasks[childID]}
	spaceRepo.spaces[spaceID] = &models.Space{ID: spaceID, DefaultLaborRate: makeRate("50.00")}
	stationRepo.stations[spaceID] = nil

	base := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	timeRepo.events[childID] = []models.TimeEvent{
		makeTimeEvent(childID, models.TimeEventTypeCheckIn, base),
		makeTimeEvent(childID, models.TimeEventTypeCheckOut, base.Add(1*time.Hour)),
	}

	result, err := svc.GetJobCostSummary(jobID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Estimated should be zero.
	if !result.Estimated.LaborHours.IsZero() {
		t.Errorf("expected zero estimated hours, got %s", result.Estimated.LaborHours)
	}

	// Actual should be 1h.
	expectedActHours := decimal.NewFromInt(1)
	if !result.Actual.LaborHours.Equal(expectedActHours) {
		t.Errorf("expected 1h actual, got %s", result.Actual.LaborHours)
	}

	// Variance percent should be zero (division by zero protection).
	if !result.Variance.Percent.IsZero() {
		t.Errorf("expected zero variance percent, got %s", result.Variance.Percent)
	}
}

func TestGetJobCostSummary_NoActivity(t *testing.T) {
	svc, _, _, taskRepo, spaceRepo, stationRepo := newTestCostServiceWithStations()

	spaceID := uuid.New()
	jobID := "job-1"

	est := 7200 // 2 hours
	taskRepo.tasks[jobID] = &models.Task{ID: jobID, SpaceID: spaceID, Type: models.TaskTypeJob, EstimatedTimeSecs: &est}
	taskRepo.descendants[jobID] = nil
	spaceRepo.spaces[spaceID] = &models.Space{ID: spaceID, DefaultLaborRate: makeRate("40.00")}
	stationRepo.stations[spaceID] = nil

	result, err := svc.GetJobCostSummary(jobID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Estimated: 2h × $40 = $80
	expectedEstCost := decimal.NewFromFloat(80.00)
	if !result.Estimated.LaborCost.Equal(expectedEstCost) {
		t.Errorf("expected estimated cost %s, got %s", expectedEstCost, result.Estimated.LaborCost)
	}

	// Actual should be zero.
	if !result.Actual.LaborCost.IsZero() {
		t.Errorf("expected zero actual cost, got %s", result.Actual.LaborCost)
	}

	// Variance: 0 - 80 = -80 (under budget)
	expectedVarCost := decimal.NewFromFloat(-80.00)
	if !result.Variance.LaborCost.Equal(expectedVarCost) {
		t.Errorf("expected variance cost %s, got %s", expectedVarCost, result.Variance.LaborCost)
	}
}

func TestGetJobCostSummary_NoLaborRate(t *testing.T) {
	svc, _, _, taskRepo, spaceRepo, _ := newTestCostServiceWithStations()

	spaceID := uuid.New()
	jobID := "job-1"
	taskRepo.tasks[jobID] = &models.Task{ID: jobID, SpaceID: spaceID}
	spaceRepo.spaces[spaceID] = &models.Space{ID: spaceID, DefaultLaborRate: nil}

	_, err := svc.GetJobCostSummary(jobID)
	if err == nil {
		t.Fatal("expected error for missing labor rate")
	}
}
