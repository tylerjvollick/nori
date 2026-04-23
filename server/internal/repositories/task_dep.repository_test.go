package repositories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

// setupTaskDepTestDB sets up a test database and cleans task-related tables.
func setupTaskDepTestDB(t *testing.T) *gorm.DB {
	db := setupTestDB(t)

	// Clean task_dep before task (FK constraint)
	db.Exec("DELETE FROM task_dep")
	db.Exec("DELETE FROM task")

	return db
}

// createTestTask creates a task for testing. Requires a space and user to exist.
func createTestTask(t *testing.T, db *gorm.DB, id string, spaceID, createdByID uuid.UUID) *models.Task {
	task := &models.Task{
		ID:          id,
		SpaceID:     spaceID,
		CreatedByID: createdByID,
		Title:       "Test Task " + id,
		Type:        models.TaskTypeTask,
		Status:      models.TaskStatusOpen,
	}
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("Failed to create test task %s: %v", id, err)
	}
	return task
}

func TestTaskDepRepository_AddDep(t *testing.T) {
	db := setupTaskDepTestDB(t)
	repo := NewTaskDepRepository(db)

	account := createTestAccount(t, db)
	user := createTestUser(t, db, "taskdep-test@example.com", account.ID)
	space := createTestSpace(t, db, "TaskDep Space", account.ID)

	taskA := createTestTask(t, db, "tdep-a", space.ID, user.ID)
	taskB := createTestTask(t, db, "tdep-b", space.ID, user.ID)

	dep := &models.TaskDep{
		ID:         uuid.New(),
		FromTaskID: taskA.ID,
		ToTaskID:   taskB.ID,
		Type:       models.DepTypeBlocks,
	}

	err := repo.AddDep(dep)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify it was created
	var retrieved models.TaskDep
	err = db.First(&retrieved, "id = ?", dep.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve created dep: %v", err)
	}
	if retrieved.FromTaskID != taskA.ID {
		t.Errorf("Expected FromTaskID %q, got %q", taskA.ID, retrieved.FromTaskID)
	}
	if retrieved.ToTaskID != taskB.ID {
		t.Errorf("Expected ToTaskID %q, got %q", taskB.ID, retrieved.ToTaskID)
	}
	if retrieved.Type != models.DepTypeBlocks {
		t.Errorf("Expected type %q, got %q", models.DepTypeBlocks, retrieved.Type)
	}
}

func TestTaskDepRepository_AddDep_SelfReference(t *testing.T) {
	db := setupTaskDepTestDB(t)
	repo := NewTaskDepRepository(db)

	account := createTestAccount(t, db)
	user := createTestUser(t, db, "taskdep-self@example.com", account.ID)
	space := createTestSpace(t, db, "Self Ref Space", account.ID)

	taskA := createTestTask(t, db, "self-a", space.ID, user.ID)

	dep := &models.TaskDep{
		ID:         uuid.New(),
		FromTaskID: taskA.ID,
		ToTaskID:   taskA.ID,
		Type:       models.DepTypeBlocks,
	}

	err := repo.AddDep(dep)
	if err == nil {
		t.Fatal("Expected error for self-referential dependency, got nil")
	}
}

func TestTaskDepRepository_AddDep_DuplicateRejected(t *testing.T) {
	db := setupTaskDepTestDB(t)
	repo := NewTaskDepRepository(db)

	account := createTestAccount(t, db)
	user := createTestUser(t, db, "taskdep-dup@example.com", account.ID)
	space := createTestSpace(t, db, "Dup Space", account.ID)

	taskA := createTestTask(t, db, "dup-a", space.ID, user.ID)
	taskB := createTestTask(t, db, "dup-b", space.ID, user.ID)

	dep1 := &models.TaskDep{
		ID:         uuid.New(),
		FromTaskID: taskA.ID,
		ToTaskID:   taskB.ID,
		Type:       models.DepTypeBlocks,
	}
	err := repo.AddDep(dep1)
	if err != nil {
		t.Fatalf("Expected no error on first add, got: %v", err)
	}

	dep2 := &models.TaskDep{
		ID:         uuid.New(),
		FromTaskID: taskA.ID,
		ToTaskID:   taskB.ID,
		Type:       models.DepTypeWaitsFor,
	}
	err = repo.AddDep(dep2)
	if err == nil {
		t.Fatal("Expected error for duplicate edge, got nil")
	}
}

func TestTaskDepRepository_AddDep_CycleDetected(t *testing.T) {
	db := setupTaskDepTestDB(t)
	repo := NewTaskDepRepository(db)

	account := createTestAccount(t, db)
	user := createTestUser(t, db, "taskdep-cycle@example.com", account.ID)
	space := createTestSpace(t, db, "Cycle Space", account.ID)

	taskA := createTestTask(t, db, "cyc-a", space.ID, user.ID)
	taskB := createTestTask(t, db, "cyc-b", space.ID, user.ID)
	taskC := createTestTask(t, db, "cyc-c", space.ID, user.ID)

	// A depends on B (A → B)
	err := repo.AddDep(&models.TaskDep{
		ID:         uuid.New(),
		FromTaskID: taskA.ID,
		ToTaskID:   taskB.ID,
		Type:       models.DepTypeBlocks,
	})
	if err != nil {
		t.Fatalf("Failed to add A→B: %v", err)
	}

	// B depends on C (B → C)
	err = repo.AddDep(&models.TaskDep{
		ID:         uuid.New(),
		FromTaskID: taskB.ID,
		ToTaskID:   taskC.ID,
		Type:       models.DepTypeBlocks,
	})
	if err != nil {
		t.Fatalf("Failed to add B→C: %v", err)
	}

	// C depends on A (C → A) should be rejected — creates cycle A→B→C→A
	err = repo.AddDep(&models.TaskDep{
		ID:         uuid.New(),
		FromTaskID: taskC.ID,
		ToTaskID:   taskA.ID,
		Type:       models.DepTypeBlocks,
	})
	if err == nil {
		t.Fatal("Expected error for cycle C→A, got nil")
	}
}

func TestTaskDepRepository_AddDep_DirectCycleDetected(t *testing.T) {
	db := setupTaskDepTestDB(t)
	repo := NewTaskDepRepository(db)

	account := createTestAccount(t, db)
	user := createTestUser(t, db, "taskdep-dcycle@example.com", account.ID)
	space := createTestSpace(t, db, "Direct Cycle Space", account.ID)

	taskA := createTestTask(t, db, "dc-a", space.ID, user.ID)
	taskB := createTestTask(t, db, "dc-b", space.ID, user.ID)

	// A depends on B
	err := repo.AddDep(&models.TaskDep{
		ID:         uuid.New(),
		FromTaskID: taskA.ID,
		ToTaskID:   taskB.ID,
		Type:       models.DepTypeBlocks,
	})
	if err != nil {
		t.Fatalf("Failed to add A→B: %v", err)
	}

	// B depends on A — direct cycle
	err = repo.AddDep(&models.TaskDep{
		ID:         uuid.New(),
		FromTaskID: taskB.ID,
		ToTaskID:   taskA.ID,
		Type:       models.DepTypeBlocks,
	})
	if err == nil {
		t.Fatal("Expected error for direct cycle B→A, got nil")
	}
}

func TestTaskDepRepository_RemoveDep(t *testing.T) {
	db := setupTaskDepTestDB(t)
	repo := NewTaskDepRepository(db)

	account := createTestAccount(t, db)
	user := createTestUser(t, db, "taskdep-rm@example.com", account.ID)
	space := createTestSpace(t, db, "Remove Space", account.ID)

	taskA := createTestTask(t, db, "rm-a", space.ID, user.ID)
	taskB := createTestTask(t, db, "rm-b", space.ID, user.ID)

	dep := &models.TaskDep{
		ID:         uuid.New(),
		FromTaskID: taskA.ID,
		ToTaskID:   taskB.ID,
		Type:       models.DepTypeBlocks,
	}
	err := repo.AddDep(dep)
	if err != nil {
		t.Fatalf("Failed to add dep: %v", err)
	}

	err = repo.RemoveDep(taskA.ID, taskB.ID)
	if err != nil {
		t.Fatalf("Expected no error on remove, got: %v", err)
	}

	// Verify it was deleted
	var count int64
	db.Model(&models.TaskDep{}).Where("from_task_id = ? AND to_task_id = ?", taskA.ID, taskB.ID).Count(&count)
	if count != 0 {
		t.Errorf("Expected dep to be deleted, found %d", count)
	}
}

func TestTaskDepRepository_RemoveDep_NotFound(t *testing.T) {
	db := setupTaskDepTestDB(t)
	repo := NewTaskDepRepository(db)

	err := repo.RemoveDep("nonexistent-1", "nonexistent-2")
	if err == nil {
		t.Fatal("Expected error for removing nonexistent dep, got nil")
	}
}

func TestTaskDepRepository_GetBlockers(t *testing.T) {
	db := setupTaskDepTestDB(t)
	repo := NewTaskDepRepository(db)

	account := createTestAccount(t, db)
	user := createTestUser(t, db, "taskdep-blockers@example.com", account.ID)
	space := createTestSpace(t, db, "Blockers Space", account.ID)

	taskA := createTestTask(t, db, "blk-a", space.ID, user.ID)
	taskB := createTestTask(t, db, "blk-b", space.ID, user.ID)
	taskC := createTestTask(t, db, "blk-c", space.ID, user.ID)

	// B depends on A: FromTaskID=B (blocked), ToTaskID=A (blocker)
	// C depends on A: FromTaskID=C (blocked), ToTaskID=A (blocker)
	// GetBlockers(B) returns deps where from_task_id=B AND type=blocks
	// → returns the B→A dep (A blocks B)

	err := repo.AddDep(&models.TaskDep{
		ID: uuid.New(), FromTaskID: taskB.ID, ToTaskID: taskA.ID, Type: models.DepTypeBlocks,
	})
	if err != nil {
		t.Fatalf("Failed to add B→A: %v", err)
	}

	err = repo.AddDep(&models.TaskDep{
		ID: uuid.New(), FromTaskID: taskC.ID, ToTaskID: taskA.ID, Type: models.DepTypeBlocks,
	})
	if err != nil {
		t.Fatalf("Failed to add C→A: %v", err)
	}

	// Also add a non-blocking dep — should not appear in GetBlockers
	err = repo.AddDep(&models.TaskDep{
		ID: uuid.New(), FromTaskID: taskB.ID, ToTaskID: taskC.ID, Type: models.DepTypeRelated,
	})
	if err != nil {
		t.Fatalf("Failed to add B→C (related): %v", err)
	}

	// GetBlockers(B) should return 1 dep: B→A (A blocks B)
	blockers, err := repo.GetBlockers(taskB.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(blockers) != 1 {
		t.Fatalf("Expected 1 blocker for B, got %d", len(blockers))
	}
	if blockers[0].FromTaskID != taskB.ID {
		t.Errorf("Expected FromTaskID %q, got %q", taskB.ID, blockers[0].FromTaskID)
	}
	if blockers[0].ToTaskID != taskA.ID {
		t.Errorf("Expected ToTaskID (blocker) %q, got %q", taskA.ID, blockers[0].ToTaskID)
	}

	// GetBlockers(C) should return 1 dep: C→A (A blocks C)
	blockersC, err := repo.GetBlockers(taskC.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(blockersC) != 1 {
		t.Fatalf("Expected 1 blocker for C, got %d", len(blockersC))
	}

	// GetBlockers(A) should return 0 — nothing blocks A
	blockersA, err := repo.GetBlockers(taskA.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(blockersA) != 0 {
		t.Fatalf("Expected 0 blockers for A, got %d", len(blockersA))
	}
}

func TestTaskDepRepository_GetDependents(t *testing.T) {
	db := setupTaskDepTestDB(t)
	repo := NewTaskDepRepository(db)

	account := createTestAccount(t, db)
	user := createTestUser(t, db, "taskdep-deps@example.com", account.ID)
	space := createTestSpace(t, db, "Dependents Space", account.ID)

	taskA := createTestTask(t, db, "dep-a", space.ID, user.ID)
	taskB := createTestTask(t, db, "dep-b", space.ID, user.ID)
	taskC := createTestTask(t, db, "dep-c", space.ID, user.ID)

	// B depends on A (A blocks B): FromTaskID=B (blocked), ToTaskID=A (blocker)
	err := repo.AddDep(&models.TaskDep{
		ID: uuid.New(), FromTaskID: taskB.ID, ToTaskID: taskA.ID, Type: models.DepTypeBlocks,
	})
	if err != nil {
		t.Fatalf("Failed to add B→A: %v", err)
	}

	// C depends on A (A blocks C): FromTaskID=C (blocked), ToTaskID=A (blocker)
	err = repo.AddDep(&models.TaskDep{
		ID: uuid.New(), FromTaskID: taskC.ID, ToTaskID: taskA.ID, Type: models.DepTypeWaitsFor,
	})
	if err != nil {
		t.Fatalf("Failed to add C→A: %v", err)
	}

	// GetDependents(A) should return 2 deps — B and C both depend on A
	dependents, err := repo.GetDependents(taskA.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(dependents) != 2 {
		t.Fatalf("Expected 2 dependents, got %d", len(dependents))
	}

	for _, d := range dependents {
		if d.ToTaskID != taskA.ID {
			t.Errorf("Expected ToTaskID %q, got %q", taskA.ID, d.ToTaskID)
		}
	}

	// GetDependents(B) should return 0 — nothing depends on B
	dependentsB, err := repo.GetDependents(taskB.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(dependentsB) != 0 {
		t.Fatalf("Expected 0 dependents for B, got %d", len(dependentsB))
	}
}

func TestTaskDepRepository_GetAllForTask(t *testing.T) {
	db := setupTaskDepTestDB(t)
	repo := NewTaskDepRepository(db)

	account := createTestAccount(t, db)
	user := createTestUser(t, db, "taskdep-all@example.com", account.ID)
	space := createTestSpace(t, db, "All Space", account.ID)

	taskA := createTestTask(t, db, "all-a", space.ID, user.ID)
	taskB := createTestTask(t, db, "all-b", space.ID, user.ID)
	taskC := createTestTask(t, db, "all-c", space.ID, user.ID)

	// A→B (A depends on B)
	err := repo.AddDep(&models.TaskDep{
		ID: uuid.New(), FromTaskID: taskA.ID, ToTaskID: taskB.ID, Type: models.DepTypeBlocks,
	})
	if err != nil {
		t.Fatalf("Failed to add A→B: %v", err)
	}

	// C→A (C depends on A)
	err = repo.AddDep(&models.TaskDep{
		ID: uuid.New(), FromTaskID: taskC.ID, ToTaskID: taskA.ID, Type: models.DepTypeWaitsFor,
	})
	if err != nil {
		t.Fatalf("Failed to add C→A: %v", err)
	}

	// B→C (B depends on C) — should NOT show up for task A
	err = repo.AddDep(&models.TaskDep{
		ID: uuid.New(), FromTaskID: taskB.ID, ToTaskID: taskC.ID, Type: models.DepTypeRelated,
	})
	if err != nil {
		t.Fatalf("Failed to add B→C: %v", err)
	}

	allDeps, err := repo.GetAllForTask(taskA.ID)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(allDeps) != 2 {
		t.Fatalf("Expected 2 deps for task A, got %d", len(allDeps))
	}
}

func TestTaskDepRepository_GetBlockers_Empty(t *testing.T) {
	db := setupTaskDepTestDB(t)
	repo := NewTaskDepRepository(db)

	blockers, err := repo.GetBlockers("nonexistent")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(blockers) != 0 {
		t.Errorf("Expected 0 blockers, got %d", len(blockers))
	}
}
