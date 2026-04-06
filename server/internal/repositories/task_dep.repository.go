package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type TaskDepRepository struct {
	db *gorm.DB
}

func NewTaskDepRepository(db *gorm.DB) *TaskDepRepository {
	return &TaskDepRepository{db: db}
}

// WithTx returns a new TaskDepRepository that uses the given transaction handle.
func (r *TaskDepRepository) WithTx(tx *gorm.DB) *TaskDepRepository {
	return &TaskDepRepository{db: tx}
}

// AddDep creates a new dependency edge between two tasks.
// It rejects the dependency if it would create a cycle in the dependency graph.
func (r *TaskDepRepository) AddDep(dep *models.TaskDep) error {
	if dep.FromTaskID == dep.ToTaskID {
		return fmt.Errorf("a task cannot depend on itself")
	}

	// Check for cycles: if we can reach FromTaskID starting from ToTaskID
	// by following existing edges (ToTaskID → ... → FromTaskID), then adding
	// this edge would create a cycle.
	if err := r.detectCycle(dep.FromTaskID, dep.ToTaskID); err != nil {
		return err
	}

	return r.db.Create(dep).Error
}

// detectCycle checks whether adding an edge from→to would create a cycle.
// It does a BFS from `to` following existing dependency edges to see if
// `from` is reachable. If so, the new edge would close a cycle.
func (r *TaskDepRepository) detectCycle(from, to string) error {
	visited := map[string]bool{}
	queue := []string{to}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == from {
			return fmt.Errorf("adding dependency from %q to %q would create a cycle", from, to)
		}

		if visited[current] {
			continue
		}
		visited[current] = true

		// Follow edges where current task depends on other tasks
		// (current is FromTaskID, follow to ToTaskID)
		var deps []models.TaskDep
		if err := r.db.Where("from_task_id = ?", current).Find(&deps).Error; err != nil {
			return fmt.Errorf("cycle detection query failed: %w", err)
		}
		for _, d := range deps {
			if !visited[d.ToTaskID] {
				queue = append(queue, d.ToTaskID)
			}
		}
	}

	return nil
}

// RemoveDep deletes the dependency edge between two specific tasks.
func (r *TaskDepRepository) RemoveDep(fromID, toID string) error {
	result := r.db.Where("from_task_id = ? AND to_task_id = ?", fromID, toID).
		Delete(&models.TaskDep{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("dependency from %q to %q not found", fromID, toID)
	}
	return nil
}

// GetBlockers returns deps where the given task is the target (ToTaskID)
// and the dependency type is "blocks". These are the tasks blocking this one.
func (r *TaskDepRepository) GetBlockers(taskID string) ([]models.TaskDep, error) {
	var deps []models.TaskDep
	err := r.db.Where("to_task_id = ? AND type = ?", taskID, models.DepTypeBlocks).
		Find(&deps).Error
	return deps, err
}

// GetDependents returns deps where the given task is the source (FromTaskID).
// These are the tasks that depend on this task.
func (r *TaskDepRepository) GetDependents(taskID string) ([]models.TaskDep, error) {
	var deps []models.TaskDep
	err := r.db.Where("from_task_id = ?", taskID).
		Find(&deps).Error
	return deps, err
}

// RemoveDepByID deletes a dependency edge by its UUID primary key.
func (r *TaskDepRepository) RemoveDepByID(id uuid.UUID) error {
	result := r.db.Where("id = ?", id).Delete(&models.TaskDep{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("dependency %q not found", id)
	}
	return nil
}

// GetAllForTask returns all dependency edges where the given task
// appears as either source or target.
func (r *TaskDepRepository) GetAllForTask(taskID string) ([]models.TaskDep, error) {
	var deps []models.TaskDep
	err := r.db.Where("from_task_id = ? OR to_task_id = ?", taskID, taskID).
		Find(&deps).Error
	return deps, err
}
