package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/tylerjvollick/nori/internal/models"
)

// CostEntryRepositoryInterface defines the methods needed from a cost entry repository.
type CostEntryRepositoryInterface interface {
	Create(entry *models.CostEntry) error
	GetByID(id uuid.UUID) (*models.CostEntry, error)
	GetByTaskID(taskID string) ([]models.CostEntry, error)
	Delete(id uuid.UUID) error
	DeleteByTaskIDAndCostType(taskID string, costType models.CostType) error
}

// CostTimeEventRepositoryInterface defines the methods needed to query time events.
type CostTimeEventRepositoryInterface interface {
	GetByTaskID(taskID string) ([]models.TimeEvent, error)
}

// CostTaskRepositoryInterface defines the methods needed to query tasks.
type CostTaskRepositoryInterface interface {
	GetByID(id string) (*models.Task, error)
	GetDescendants(idPrefix string) ([]models.Task, error)
}

// CostSpaceRepositoryInterface defines the methods needed to query spaces.
type CostSpaceRepositoryInterface interface {
	FindByID(id uuid.UUID) (*models.Space, error)
}

// CostStationRepositoryInterface defines the methods needed to query stations.
type CostStationRepositoryInterface interface {
	GetBySpaceID(spaceID uuid.UUID) ([]models.Station, error)
}

// CostService computes and stores cost entries for tasks and jobs.
type CostService struct {
	costEntryRepo CostEntryRepositoryInterface
	timeEventRepo CostTimeEventRepositoryInterface
	taskRepo      CostTaskRepositoryInterface
	spaceRepo     CostSpaceRepositoryInterface
	stationRepo   CostStationRepositoryInterface
}

// NewCostService creates a new CostService with injected dependencies.
func NewCostService(
	costEntryRepo CostEntryRepositoryInterface,
	timeEventRepo CostTimeEventRepositoryInterface,
	taskRepo CostTaskRepositoryInterface,
	spaceRepo CostSpaceRepositoryInterface,
	stationRepo CostStationRepositoryInterface,
) *CostService {
	return &CostService{
		costEntryRepo: costEntryRepo,
		timeEventRepo: timeEventRepo,
		taskRepo:      taskRepo,
		spaceRepo:     spaceRepo,
		stationRepo:   stationRepo,
	}
}

// LaborCostResult holds the computed labor cost for a single task.
type LaborCostResult struct {
	TaskID       string          `json:"taskId"`
	ActiveSecs   int64           `json:"activeSeconds"`
	HourlyRate   decimal.Decimal `json:"hourlyRate"`
	LaborCost    decimal.Decimal `json:"laborCost"`
	CostEntryID  uuid.UUID       `json:"costEntryId"`
}

// JobLaborCostResult holds the aggregated labor cost for a job and its children.
type JobLaborCostResult struct {
	JobID        string            `json:"jobId"`
	TotalCost    decimal.Decimal   `json:"totalCost"`
	TotalSecs    int64             `json:"totalSeconds"`
	HourlyRate   decimal.Decimal   `json:"hourlyRate"`
	TaskResults  []LaborCostResult `json:"taskResults"`
}

// EstimatedLaborCost holds the estimated labor cost from recipe step times.
type EstimatedLaborCost struct {
	TotalEstimatedSecs int64           `json:"totalEstimatedSeconds"`
	HourlyRate         decimal.Decimal `json:"hourlyRate"`
	EstimatedCost      decimal.Decimal `json:"estimatedCost"`
}

// ComputeTaskLaborCost computes and stores the labor cost for a single task
// based on its TimeEvents and the space's default labor rate.
// It replaces any existing labor cost entry for the task.
func (s *CostService) ComputeTaskLaborCost(taskID string, createdByID uuid.UUID) (*LaborCostResult, error) {
	task, err := s.taskRepo.GetByID(taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	space, err := s.spaceRepo.FindByID(task.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("space not found: %w", err)
	}

	if space.DefaultLaborRate == nil {
		return nil, fmt.Errorf("space %q has no default labor rate configured", space.ID)
	}

	events, err := s.timeEventRepo.GetByTaskID(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch time events: %w", err)
	}

	activeSecs := computeActiveSeconds(events)

	hours := decimal.NewFromInt(activeSecs).Div(decimal.NewFromInt(3600))
	laborCost := hours.Mul(*space.DefaultLaborRate)

	// Remove any existing labor cost entry for this task before creating a new one.
	if err := s.costEntryRepo.DeleteByTaskIDAndCostType(taskID, models.CostTypeLabor); err != nil {
		return nil, fmt.Errorf("failed to clear existing labor cost: %w", err)
	}

	hoursUnit := "hours"
	entry := &models.CostEntry{
		TaskID:      taskID,
		CostType:    models.CostTypeLabor,
		Description: "Labor cost computed from time events",
		Amount:      laborCost,
		Quantity:    &hours,
		Unit:        &hoursUnit,
		UnitCost:    space.DefaultLaborRate,
		CreatedByID: createdByID,
	}

	if err := s.costEntryRepo.Create(entry); err != nil {
		return nil, fmt.Errorf("failed to create cost entry: %w", err)
	}

	return &LaborCostResult{
		TaskID:      taskID,
		ActiveSecs:  activeSecs,
		HourlyRate:  *space.DefaultLaborRate,
		LaborCost:   laborCost,
		CostEntryID: entry.ID,
	}, nil
}

// ComputeJobLaborCost computes and stores labor costs for a job and all its
// descendant tasks. Returns the aggregated result.
func (s *CostService) ComputeJobLaborCost(jobID string, createdByID uuid.UUID) (*JobLaborCostResult, error) {
	job, err := s.taskRepo.GetByID(jobID)
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}

	space, err := s.spaceRepo.FindByID(job.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("space not found: %w", err)
	}

	if space.DefaultLaborRate == nil {
		return nil, fmt.Errorf("space %q has no default labor rate configured", space.ID)
	}

	descendants, err := s.taskRepo.GetDescendants(jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch descendants: %w", err)
	}

	// Collect all task IDs (job root + descendants).
	allTasks := make([]models.Task, 0, len(descendants)+1)
	allTasks = append(allTasks, *job)
	allTasks = append(allTasks, descendants...)

	var results []LaborCostResult
	totalCost := decimal.Zero
	var totalSecs int64

	for _, task := range allTasks {
		events, err := s.timeEventRepo.GetByTaskID(task.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch time events for task %q: %w", task.ID, err)
		}

		activeSecs := computeActiveSeconds(events)
		if activeSecs == 0 {
			continue
		}

		hours := decimal.NewFromInt(activeSecs).Div(decimal.NewFromInt(3600))
		laborCost := hours.Mul(*space.DefaultLaborRate)

		// Replace existing labor cost for this task.
		if err := s.costEntryRepo.DeleteByTaskIDAndCostType(task.ID, models.CostTypeLabor); err != nil {
			return nil, fmt.Errorf("failed to clear labor cost for task %q: %w", task.ID, err)
		}

		hoursUnit := "hours"
		entry := &models.CostEntry{
			TaskID:      task.ID,
			CostType:    models.CostTypeLabor,
			Description: "Labor cost computed from time events",
			Amount:      laborCost,
			Quantity:    &hours,
			Unit:        &hoursUnit,
			UnitCost:    space.DefaultLaborRate,
			CreatedByID: createdByID,
		}

		if err := s.costEntryRepo.Create(entry); err != nil {
			return nil, fmt.Errorf("failed to create cost entry for task %q: %w", task.ID, err)
		}

		results = append(results, LaborCostResult{
			TaskID:      task.ID,
			ActiveSecs:  activeSecs,
			HourlyRate:  *space.DefaultLaborRate,
			LaborCost:   laborCost,
			CostEntryID: entry.ID,
		})

		totalCost = totalCost.Add(laborCost)
		totalSecs += activeSecs
	}

	return &JobLaborCostResult{
		JobID:       jobID,
		TotalCost:   totalCost,
		TotalSecs:   totalSecs,
		HourlyRate:  *space.DefaultLaborRate,
		TaskResults: results,
	}, nil
}

// EstimateRecipeLaborCost computes the estimated labor cost for a recipe by
// summing EstimatedTimeSecs across all tasks in the tree and multiplying by
// the space's default labor rate.
func (s *CostService) EstimateRecipeLaborCost(rootTaskID string, spaceID uuid.UUID) (*EstimatedLaborCost, error) {
	space, err := s.spaceRepo.FindByID(spaceID)
	if err != nil {
		return nil, fmt.Errorf("space not found: %w", err)
	}

	if space.DefaultLaborRate == nil {
		return nil, fmt.Errorf("space %q has no default labor rate configured", space.ID)
	}

	rootTask, err := s.taskRepo.GetByID(rootTaskID)
	if err != nil {
		return nil, fmt.Errorf("root task not found: %w", err)
	}

	descendants, err := s.taskRepo.GetDescendants(rootTaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch descendants: %w", err)
	}

	allTasks := make([]models.Task, 0, len(descendants)+1)
	allTasks = append(allTasks, *rootTask)
	allTasks = append(allTasks, descendants...)

	var totalSecs int64
	for _, task := range allTasks {
		if task.EstimatedTimeSecs != nil {
			totalSecs += int64(*task.EstimatedTimeSecs)
		}
	}

	hours := decimal.NewFromInt(totalSecs).Div(decimal.NewFromInt(3600))
	estimatedCost := hours.Mul(*space.DefaultLaborRate)

	return &EstimatedLaborCost{
		TotalEstimatedSecs: totalSecs,
		HourlyRate:         *space.DefaultLaborRate,
		EstimatedCost:      estimatedCost,
	}, nil
}

// GetTaskCosts returns all cost entries for a task.
func (s *CostService) GetTaskCosts(taskID string) ([]models.CostEntry, error) {
	return s.costEntryRepo.GetByTaskID(taskID)
}

// CostSummary holds the aggregated cost summary for a job.
type CostSummary struct {
	JobID    string              `json:"jobId"`
	Estimated CostBreakdown     `json:"estimated"`
	Actual    CostBreakdown     `json:"actual"`
	Variance  CostVariance      `json:"variance"`
	ByStation []StationCostSummary `json:"byStation"`
}

// CostBreakdown holds labor hours, cost, and total for estimated or actual.
type CostBreakdown struct {
	LaborHours decimal.Decimal `json:"laborHours"`
	LaborCost  decimal.Decimal `json:"laborCost"`
	Total      decimal.Decimal `json:"total"`
}

// CostVariance holds the difference between actual and estimated.
type CostVariance struct {
	LaborHours decimal.Decimal `json:"laborHours"`
	LaborCost  decimal.Decimal `json:"laborCost"`
	Total      decimal.Decimal `json:"total"`
	Percent    decimal.Decimal `json:"percent"`
}

// StationCostSummary holds per-station cost breakdown.
type StationCostSummary struct {
	StationID      string          `json:"stationId"`
	StationName    string          `json:"stationName"`
	EstimatedHours decimal.Decimal `json:"estimatedHours"`
	ActualHours    decimal.Decimal `json:"actualHours"`
	Variance       decimal.Decimal `json:"variance"`
}

// GetJobCostSummary returns the cost summary for a job, comparing estimated
// (from recipe estimated_time_secs) vs actual (from TimeEvents), broken down
// by station.
func (s *CostService) GetJobCostSummary(jobID string) (*CostSummary, error) {
	job, err := s.taskRepo.GetByID(jobID)
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}

	space, err := s.spaceRepo.FindByID(job.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("space not found: %w", err)
	}

	if space.DefaultLaborRate == nil {
		return nil, fmt.Errorf("space %q has no default labor rate configured", space.ID)
	}
	hourlyRate := *space.DefaultLaborRate

	descendants, err := s.taskRepo.GetDescendants(jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch descendants: %w", err)
	}

	allTasks := make([]models.Task, 0, len(descendants)+1)
	allTasks = append(allTasks, *job)
	allTasks = append(allTasks, descendants...)

	// Load stations for name lookup.
	stations, err := s.stationRepo.GetBySpaceID(job.SpaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stations: %w", err)
	}
	stationNames := make(map[uuid.UUID]string, len(stations))
	for _, st := range stations {
		stationNames[st.ID] = st.Name
	}

	// Aggregate estimated and actual hours per station.
	type stationAccum struct {
		estimatedSecs int64
		actualSecs    int64
	}
	byStation := make(map[uuid.UUID]*stationAccum)

	var totalEstimatedSecs int64
	var totalActualSecs int64

	for _, task := range allTasks {
		// Estimated time from recipe step.
		var estSecs int64
		if task.EstimatedTimeSecs != nil {
			estSecs = int64(*task.EstimatedTimeSecs)
			totalEstimatedSecs += estSecs
		}

		// Actual time from time events.
		events, err := s.timeEventRepo.GetByTaskID(task.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch time events for task %q: %w", task.ID, err)
		}
		actSecs := computeActiveSeconds(events)
		totalActualSecs += actSecs

		// Accumulate by station.
		if task.StationID != nil {
			acc, ok := byStation[*task.StationID]
			if !ok {
				acc = &stationAccum{}
				byStation[*task.StationID] = acc
			}
			acc.estimatedSecs += estSecs
			acc.actualSecs += actSecs
		}
	}

	secsToHours := func(secs int64) decimal.Decimal {
		return decimal.NewFromInt(secs).Div(decimal.NewFromInt(3600))
	}

	estHours := secsToHours(totalEstimatedSecs)
	estCost := estHours.Mul(hourlyRate)
	actHours := secsToHours(totalActualSecs)
	actCost := actHours.Mul(hourlyRate)

	varHours := actHours.Sub(estHours)
	varCost := actCost.Sub(estCost)

	var varPercent decimal.Decimal
	if !estCost.IsZero() {
		varPercent = varCost.Div(estCost).Mul(decimal.NewFromInt(100))
	}

	// Build by-station breakdown.
	stationSummaries := make([]StationCostSummary, 0, len(byStation))
	for stID, acc := range byStation {
		estH := secsToHours(acc.estimatedSecs)
		actH := secsToHours(acc.actualSecs)
		stationSummaries = append(stationSummaries, StationCostSummary{
			StationID:      stID.String(),
			StationName:    stationNames[stID],
			EstimatedHours: estH,
			ActualHours:    actH,
			Variance:       actH.Sub(estH),
		})
	}

	return &CostSummary{
		JobID: jobID,
		Estimated: CostBreakdown{
			LaborHours: estHours,
			LaborCost:  estCost,
			Total:      estCost,
		},
		Actual: CostBreakdown{
			LaborHours: actHours,
			LaborCost:  actCost,
			Total:      actCost,
		},
		Variance: CostVariance{
			LaborHours: varHours,
			LaborCost:  varCost,
			Total:      varCost,
			Percent:    varPercent,
		},
		ByStation: stationSummaries,
	}, nil
}

// computeActiveSeconds processes time events to calculate total active seconds.
// Events must be ordered by timestamp (ascending or descending — we sort internally).
// State machine: check_in → active, pause → paused, resume → active, check_out → idle.
func computeActiveSeconds(events []models.TimeEvent) int64 {
	if len(events) == 0 {
		return 0
	}

	// Sort events by timestamp ascending.
	sorted := make([]models.TimeEvent, len(events))
	copy(sorted, events)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j].Timestamp.Before(sorted[j-1].Timestamp); j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}

	var totalSecs int64
	var activeStart *time.Time

	for _, event := range sorted {
		switch event.EventType {
		case models.TimeEventTypeCheckIn, models.TimeEventTypeResume:
			if activeStart == nil {
				ts := event.Timestamp
				activeStart = &ts
			}
		case models.TimeEventTypePause, models.TimeEventTypeCheckOut:
			if activeStart != nil {
				totalSecs += int64(event.Timestamp.Sub(*activeStart).Seconds())
				activeStart = nil
			}
		}
	}

	return totalSecs
}
