package services

import (
	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

// ReadyWorkService determines which tasks are ready to be worked on,
// implementing a pull-based flow where only unblocked tasks surface.
type ReadyWorkService struct {
	db *gorm.DB
}

// NewReadyWorkService creates a new ReadyWorkService.
func NewReadyWorkService(db *gorm.DB) *ReadyWorkService {
	return &ReadyWorkService{db: db}
}

// terminalStatuses are task statuses that indicate the task is finished
// and any blocking dependency it represents is considered resolved.
var terminalStatuses = []models.TaskStatus{
	models.TaskStatusDone,
	models.TaskStatusSkipped,
	models.TaskStatusCancelled,
}

// ReadyTaskFilter holds optional filters for the ready-work query.
type ReadyTaskFilter struct {
	StationID    *uuid.UUID
	AssignedToID *uuid.UUID
}

// GetReadyTasks returns open tasks in the given space that are not blocked
// by unresolved dependencies and whose parents are also not blocked.
//
// Optional filters narrow the initial candidate set before dependency checking.
//
// Algorithm:
//  1. Find all open tasks in the space (optionally filtered by station/assignee).
//  2. Compute the blocked set: tasks that have unresolved "blocks" deps
//     (i.e., a TaskDep where Type="blocks" and the ToTask status is not terminal).
//  3. Exclude children of blocked parents (recursively — if a parent is blocked,
//     all its descendants are also blocked).
//  4. Return the remaining open tasks sorted by Priority ASC, CreatedAt ASC.
func (s *ReadyWorkService) GetReadyTasks(spaceID uuid.UUID, filter *ReadyTaskFilter) ([]models.Task, error) {
	// Step 1: Find all open tasks in the space, applying optional filters.
	var allOpen []models.Task
	query := s.db.Where("space_id = ? AND status = ?", spaceID, models.TaskStatusOpen)
	if filter != nil {
		if filter.StationID != nil {
			query = query.Where("station_id = ?", *filter.StationID)
		}
		if filter.AssignedToID != nil {
			query = query.Where("assigned_to_id = ?", *filter.AssignedToID)
		}
	}
	if err := query.Find(&allOpen).Error; err != nil {
		return nil, err
	}

	if len(allOpen) == 0 {
		return []models.Task{}, nil
	}

	// Build a lookup of open task IDs for quick membership checks.
	openTaskIDs := make([]string, len(allOpen))
	openByID := make(map[string]*models.Task, len(allOpen))
	for i := range allOpen {
		openTaskIDs[i] = allOpen[i].ID
		openByID[allOpen[i].ID] = &allOpen[i]
	}

	// Step 2: Find directly blocked tasks.
	// A task is directly blocked if it has a "blocks" dep where the blocker
	// (ToTask) is NOT in a terminal status.
	directlyBlocked, err := s.findDirectlyBlockedTasks(openTaskIDs)
	if err != nil {
		return nil, err
	}

	// Step 3: Expand the blocked set to include children of blocked parents.
	// Build parent→children mapping from the open tasks.
	blocked := s.expandBlockedToChildren(directlyBlocked, allOpen)

	// Step 4: Filter and sort.
	var ready []models.Task
	for i := range allOpen {
		if !blocked[allOpen[i].ID] {
			ready = append(ready, allOpen[i])
		}
	}

	// Sort by Priority ASC, then CreatedAt ASC.
	sortReadyTasks(ready)

	return ready, nil
}

// findDirectlyBlockedTasks returns the set of task IDs (from the given candidates)
// that have at least one unresolved "blocks" dependency.
func (s *ReadyWorkService) findDirectlyBlockedTasks(candidateIDs []string) (map[string]bool, error) {
	blocked := make(map[string]bool)

	if len(candidateIDs) == 0 {
		return blocked, nil
	}

	// Query: find task_dep rows where:
	//   - from_task_id is in our candidate set (the task that depends on something)
	//   - type = 'blocks'
	//   - the to_task (the blocker) is NOT in a terminal status
	//
	// We join task_dep with the task table on to_task_id to check the blocker's status.
	var blockedIDs []string
	err := s.db.
		Model(&models.TaskDep{}).
		Select("DISTINCT task_dep.from_task_id").
		Joins("JOIN task ON task.id = task_dep.to_task_id").
		Where("task_dep.from_task_id IN ?", candidateIDs).
		Where("task_dep.type = ?", models.DepTypeBlocks).
		Where("task.status NOT IN ?", terminalStatuses).
		Pluck("task_dep.from_task_id", &blockedIDs).Error
	if err != nil {
		return nil, err
	}

	for _, id := range blockedIDs {
		blocked[id] = true
	}

	return blocked, nil
}

// expandBlockedToChildren takes a set of directly blocked task IDs and expands
// it to include all descendants of blocked tasks. If a parent is blocked,
// all its children are also considered blocked.
func (s *ReadyWorkService) expandBlockedToChildren(directlyBlocked map[string]bool, allOpen []models.Task) map[string]bool {
	blocked := make(map[string]bool, len(directlyBlocked))
	for id := range directlyBlocked {
		blocked[id] = true
	}

	// Build parent→children index from all open tasks.
	childrenOf := make(map[string][]string)
	for _, t := range allOpen {
		if t.ParentID != nil {
			childrenOf[*t.ParentID] = append(childrenOf[*t.ParentID], t.ID)
		}
	}

	// BFS from all directly blocked tasks to propagate to children.
	queue := make([]string, 0, len(directlyBlocked))
	for id := range directlyBlocked {
		queue = append(queue, id)
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, childID := range childrenOf[current] {
			if !blocked[childID] {
				blocked[childID] = true
				queue = append(queue, childID)
			}
		}
	}

	// Also check: if a task's parent is blocked (even if the parent is not open),
	// the child should be blocked too. We need to walk up the parent chain for
	// each open task and see if any ancestor is blocked.
	// However, since we only have open tasks in allOpen, and blocked parents that
	// are in non-open statuses won't appear here, we need to also check parents
	// that might be blocked but not open. For the initial implementation, we only
	// consider the open task set (parents that are themselves open and blocked).
	// This is consistent with the beads algorithm which operates on the open set.

	return blocked
}

// sortReadyTasks sorts tasks by Priority ASC, then CreatedAt ASC.
func sortReadyTasks(tasks []models.Task) {
	// Use a simple insertion sort since the result set is typically small.
	for i := 1; i < len(tasks); i++ {
		key := tasks[i]
		j := i - 1
		for j >= 0 && taskLess(key, tasks[j]) {
			tasks[j+1] = tasks[j]
			j--
		}
		tasks[j+1] = key
	}
}

// taskLess returns true if a should sort before b.
func taskLess(a, b models.Task) bool {
	if a.Priority != b.Priority {
		return a.Priority < b.Priority
	}
	return a.CreatedAt.Before(b.CreatedAt)
}
