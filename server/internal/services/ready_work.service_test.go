package services

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// setupReadyWorkTestDB creates a test database connection and cleans relevant tables.
func setupReadyWorkTestDB(t *testing.T) *gorm.DB {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "postgres"
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "password"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "nori"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5432"
	}

	dsn := "host=" + host + " user=" + user + " password=" + password + " dbname=" + dbname + " port=" + port + " sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}

	// Clean up in dependency order (FK constraints).
	db.Exec("DELETE FROM task_dep")
	db.Exec("DELETE FROM task")
	db.Exec("DELETE FROM space_member")
	db.Exec("DELETE FROM spaces")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM accounts")

	return db
}

// rwTestAccount creates a test account with a system user.
func rwTestAccount(t *testing.T, db *gorm.DB) *models.Account {
	tempUser := &models.User{
		ID:    uuid.New(),
		Email: "rw-system-" + uuid.New().String() + "@example.com",
	}
	if err := db.Create(tempUser).Error; err != nil {
		t.Fatalf("Failed to create temp user: %v", err)
	}

	name := "RW Test Account"
	account := &models.Account{
		ID:              uuid.New(),
		Name:            &name,
		Plan:            models.Trial,
		CreatedByUserID: tempUser.ID,
	}
	if err := db.Create(account).Error; err != nil {
		t.Fatalf("Failed to create test account: %v", err)
	}
	return account
}

// rwTestUser creates a test user.
func rwTestUser(t *testing.T, db *gorm.DB, accountID uuid.UUID) *models.User {
	user := &models.User{
		ID:               uuid.New(),
		Email:            "rw-user-" + uuid.New().String() + "@example.com",
		DefaultAccountID: &accountID,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	return user
}

// rwTestSpace creates a test space.
func rwTestSpace(t *testing.T, db *gorm.DB, accountID uuid.UUID) *models.Space {
	space := &models.Space{
		ID:        uuid.New(),
		Name:      "RW Test Space",
		AccountID: accountID,
	}
	if err := db.Create(space).Error; err != nil {
		t.Fatalf("Failed to create test space: %v", err)
	}
	return space
}

// rwTestTask creates a task with the given parameters.
func rwTestTask(t *testing.T, db *gorm.DB, id string, spaceID, createdByID uuid.UUID, status models.TaskStatus, priority int, parentID *string) *models.Task {
	task := &models.Task{
		ID:          id,
		SpaceID:     spaceID,
		CreatedByID: createdByID,
		Title:       "Task " + id,
		Type:        models.TaskTypeTask,
		Status:      status,
		Priority:    priority,
		ParentID:    parentID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("Failed to create task %s: %v", id, err)
	}
	return task
}

// rwTestDep creates a blocking dependency between two tasks.
func rwTestDep(t *testing.T, db *gorm.DB, fromID, toID string) {
	dep := &models.TaskDep{
		ID:         uuid.New(),
		FromTaskID: fromID,
		ToTaskID:   toID,
		Type:       models.DepTypeBlocks,
	}
	if err := db.Create(dep).Error; err != nil {
		t.Fatalf("Failed to create dep %s -> %s: %v", fromID, toID, err)
	}
}

func TestReadyWorkService_NoTasks(t *testing.T) {
	db := setupReadyWorkTestDB(t)
	svc := NewReadyWorkService(db)

	tasks, err := svc.GetReadyTasks(uuid.New(), nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(tasks) != 0 {
		t.Errorf("Expected 0 tasks, got %d", len(tasks))
	}
}

func TestReadyWorkService_AllOpenNoDeps(t *testing.T) {
	db := setupReadyWorkTestDB(t)
	svc := NewReadyWorkService(db)

	account := rwTestAccount(t, db)
	user := rwTestUser(t, db, account.ID)
	space := rwTestSpace(t, db, account.ID)

	rwTestTask(t, db, "rw-a", space.ID, user.ID, models.TaskStatusOpen, 2, nil)
	rwTestTask(t, db, "rw-b", space.ID, user.ID, models.TaskStatusOpen, 1, nil)
	rwTestTask(t, db, "rw-c", space.ID, user.ID, models.TaskStatusOpen, 1, nil)

	tasks, err := svc.GetReadyTasks(space.ID, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("Expected 3 tasks, got %d", len(tasks))
	}

	// Verify sort order: priority ASC, then createdAt ASC.
	// rw-b and rw-c have priority 1, rw-a has priority 2.
	if tasks[0].Priority != 1 {
		t.Errorf("Expected first task priority 1, got %d", tasks[0].Priority)
	}
	if tasks[1].Priority != 1 {
		t.Errorf("Expected second task priority 1, got %d", tasks[1].Priority)
	}
	if tasks[2].Priority != 2 {
		t.Errorf("Expected third task priority 2, got %d", tasks[2].Priority)
	}
}

func TestReadyWorkService_BlockedByUnresolvedDep(t *testing.T) {
	db := setupReadyWorkTestDB(t)
	svc := NewReadyWorkService(db)

	account := rwTestAccount(t, db)
	user := rwTestUser(t, db, account.ID)
	space := rwTestSpace(t, db, account.ID)

	// Task A is open, Task B is open.
	// B depends on A (A blocks B): FromTaskID=B, ToTaskID=A, Type=blocks
	// A is open (not done), so B should be blocked.
	rwTestTask(t, db, "blocker", space.ID, user.ID, models.TaskStatusOpen, 0, nil)
	rwTestTask(t, db, "blocked", space.ID, user.ID, models.TaskStatusOpen, 0, nil)
	rwTestDep(t, db, "blocked", "blocker")

	tasks, err := svc.GetReadyTasks(space.ID, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 ready task, got %d", len(tasks))
	}
	if tasks[0].ID != "blocker" {
		t.Errorf("Expected ready task 'blocker', got %q", tasks[0].ID)
	}
}

func TestReadyWorkService_ResolvedDepUnblocks(t *testing.T) {
	db := setupReadyWorkTestDB(t)
	svc := NewReadyWorkService(db)

	account := rwTestAccount(t, db)
	user := rwTestUser(t, db, account.ID)
	space := rwTestSpace(t, db, account.ID)

	// Task A is done, Task B depends on A.
	// Since A is done, B should be unblocked and ready.
	rwTestTask(t, db, "done-blocker", space.ID, user.ID, models.TaskStatusDone, 0, nil)
	rwTestTask(t, db, "unblocked", space.ID, user.ID, models.TaskStatusOpen, 0, nil)
	rwTestDep(t, db, "unblocked", "done-blocker")

	tasks, err := svc.GetReadyTasks(space.ID, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 ready task, got %d", len(tasks))
	}
	if tasks[0].ID != "unblocked" {
		t.Errorf("Expected ready task 'unblocked', got %q", tasks[0].ID)
	}
}

func TestReadyWorkService_SkippedDepUnblocks(t *testing.T) {
	db := setupReadyWorkTestDB(t)
	svc := NewReadyWorkService(db)

	account := rwTestAccount(t, db)
	user := rwTestUser(t, db, account.ID)
	space := rwTestSpace(t, db, account.ID)

	// Blocker is skipped — should unblock the dependent.
	rwTestTask(t, db, "skipped-blocker", space.ID, user.ID, models.TaskStatusSkipped, 0, nil)
	rwTestTask(t, db, "dep-task", space.ID, user.ID, models.TaskStatusOpen, 0, nil)
	rwTestDep(t, db, "dep-task", "skipped-blocker")

	tasks, err := svc.GetReadyTasks(space.ID, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 ready task, got %d", len(tasks))
	}
	if tasks[0].ID != "dep-task" {
		t.Errorf("Expected ready task 'dep-task', got %q", tasks[0].ID)
	}
}

func TestReadyWorkService_CancelledDepUnblocks(t *testing.T) {
	db := setupReadyWorkTestDB(t)
	svc := NewReadyWorkService(db)

	account := rwTestAccount(t, db)
	user := rwTestUser(t, db, account.ID)
	space := rwTestSpace(t, db, account.ID)

	// Blocker is cancelled — should unblock the dependent.
	rwTestTask(t, db, "cancelled-blocker", space.ID, user.ID, models.TaskStatusCancelled, 0, nil)
	rwTestTask(t, db, "dep-task2", space.ID, user.ID, models.TaskStatusOpen, 0, nil)
	rwTestDep(t, db, "dep-task2", "cancelled-blocker")

	tasks, err := svc.GetReadyTasks(space.ID, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 ready task, got %d", len(tasks))
	}
	if tasks[0].ID != "dep-task2" {
		t.Errorf("Expected ready task 'dep-task2', got %q", tasks[0].ID)
	}
}

func TestReadyWorkService_ChildOfBlockedParentExcluded(t *testing.T) {
	db := setupReadyWorkTestDB(t)
	svc := NewReadyWorkService(db)

	account := rwTestAccount(t, db)
	user := rwTestUser(t, db, account.ID)
	space := rwTestSpace(t, db, account.ID)

	// Setup:
	// - prereq (open, no deps)
	// - parent (open, blocked by prereq)
	// - child (open, child of parent, no direct deps)
	//
	// child should be excluded because its parent is blocked.
	rwTestTask(t, db, "prereq", space.ID, user.ID, models.TaskStatusOpen, 0, nil)

	parentID := "parent-task"
	rwTestTask(t, db, parentID, space.ID, user.ID, models.TaskStatusOpen, 0, nil)
	rwTestDep(t, db, parentID, "prereq") // parent blocked by prereq

	rwTestTask(t, db, "child-task", space.ID, user.ID, models.TaskStatusOpen, 0, &parentID)

	tasks, err := svc.GetReadyTasks(space.ID, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Only prereq should be ready.
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 ready task, got %d", len(tasks))
	}
	if tasks[0].ID != "prereq" {
		t.Errorf("Expected ready task 'prereq', got %q", tasks[0].ID)
	}
}

func TestReadyWorkService_GrandchildOfBlockedExcluded(t *testing.T) {
	db := setupReadyWorkTestDB(t)
	svc := NewReadyWorkService(db)

	account := rwTestAccount(t, db)
	user := rwTestUser(t, db, account.ID)
	space := rwTestSpace(t, db, account.ID)

	// prereq blocks grandparent; grandparent has child; child has grandchild.
	// All descendants of grandparent should be excluded.
	rwTestTask(t, db, "gp-prereq", space.ID, user.ID, models.TaskStatusOpen, 0, nil)

	gpID := "grandparent"
	rwTestTask(t, db, gpID, space.ID, user.ID, models.TaskStatusOpen, 0, nil)
	rwTestDep(t, db, gpID, "gp-prereq")

	childID := "gp-child"
	rwTestTask(t, db, childID, space.ID, user.ID, models.TaskStatusOpen, 0, &gpID)

	rwTestTask(t, db, "gp-grandchild", space.ID, user.ID, models.TaskStatusOpen, 0, &childID)

	tasks, err := svc.GetReadyTasks(space.ID, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 ready task, got %d", len(tasks))
	}
	if tasks[0].ID != "gp-prereq" {
		t.Errorf("Expected ready task 'gp-prereq', got %q", tasks[0].ID)
	}
}

func TestReadyWorkService_MultipleBlockersAllMustResolve(t *testing.T) {
	db := setupReadyWorkTestDB(t)
	svc := NewReadyWorkService(db)

	account := rwTestAccount(t, db)
	user := rwTestUser(t, db, account.ID)
	space := rwTestSpace(t, db, account.ID)

	// Task C depends on both A and B.
	// A is done, but B is open. C should still be blocked.
	rwTestTask(t, db, "multi-a", space.ID, user.ID, models.TaskStatusDone, 0, nil)
	rwTestTask(t, db, "multi-b", space.ID, user.ID, models.TaskStatusOpen, 0, nil)
	rwTestTask(t, db, "multi-c", space.ID, user.ID, models.TaskStatusOpen, 0, nil)
	rwTestDep(t, db, "multi-c", "multi-a") // resolved (a is done)
	rwTestDep(t, db, "multi-c", "multi-b") // unresolved (b is open)

	tasks, err := svc.GetReadyTasks(space.ID, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Only multi-b should be ready (multi-c is blocked by multi-b).
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 ready task, got %d", len(tasks))
	}
	if tasks[0].ID != "multi-b" {
		t.Errorf("Expected ready task 'multi-b', got %q", tasks[0].ID)
	}
}

func TestReadyWorkService_NonBlockingDepsIgnored(t *testing.T) {
	db := setupReadyWorkTestDB(t)
	svc := NewReadyWorkService(db)

	account := rwTestAccount(t, db)
	user := rwTestUser(t, db, account.ID)
	space := rwTestSpace(t, db, account.ID)

	// Task B has a "related" dep on A (not "blocks"). B should still be ready.
	rwTestTask(t, db, "related-a", space.ID, user.ID, models.TaskStatusOpen, 0, nil)
	rwTestTask(t, db, "related-b", space.ID, user.ID, models.TaskStatusOpen, 0, nil)

	dep := &models.TaskDep{
		ID:         uuid.New(),
		FromTaskID: "related-b",
		ToTaskID:   "related-a",
		Type:       models.DepTypeRelated,
	}
	if err := db.Create(dep).Error; err != nil {
		t.Fatalf("Failed to create dep: %v", err)
	}

	tasks, err := svc.GetReadyTasks(space.ID, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("Expected 2 ready tasks, got %d", len(tasks))
	}
}

func TestReadyWorkService_OnlyOpenTasksReturned(t *testing.T) {
	db := setupReadyWorkTestDB(t)
	svc := NewReadyWorkService(db)

	account := rwTestAccount(t, db)
	user := rwTestUser(t, db, account.ID)
	space := rwTestSpace(t, db, account.ID)

	// Mix of statuses — only open tasks should appear.
	rwTestTask(t, db, "status-open", space.ID, user.ID, models.TaskStatusOpen, 0, nil)
	rwTestTask(t, db, "status-inprogress", space.ID, user.ID, models.TaskStatusInProgress, 0, nil)
	rwTestTask(t, db, "status-done", space.ID, user.ID, models.TaskStatusDone, 0, nil)
	rwTestTask(t, db, "status-skipped", space.ID, user.ID, models.TaskStatusSkipped, 0, nil)

	tasks, err := svc.GetReadyTasks(space.ID, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 ready task, got %d", len(tasks))
	}
	if tasks[0].ID != "status-open" {
		t.Errorf("Expected 'status-open', got %q", tasks[0].ID)
	}
}

func TestReadyWorkService_SpaceIsolation(t *testing.T) {
	db := setupReadyWorkTestDB(t)
	svc := NewReadyWorkService(db)

	account := rwTestAccount(t, db)
	user := rwTestUser(t, db, account.ID)
	space1 := rwTestSpace(t, db, account.ID)
	space2 := rwTestSpace(t, db, account.ID)

	rwTestTask(t, db, "space1-task", space1.ID, user.ID, models.TaskStatusOpen, 0, nil)
	rwTestTask(t, db, "space2-task", space2.ID, user.ID, models.TaskStatusOpen, 0, nil)

	tasks, err := svc.GetReadyTasks(space1.ID, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task in space1, got %d", len(tasks))
	}
	if tasks[0].ID != "space1-task" {
		t.Errorf("Expected 'space1-task', got %q", tasks[0].ID)
	}
}

func TestReadyWorkService_SortOrder(t *testing.T) {
	db := setupReadyWorkTestDB(t)
	svc := NewReadyWorkService(db)

	account := rwTestAccount(t, db)
	user := rwTestUser(t, db, account.ID)
	space := rwTestSpace(t, db, account.ID)

	// Create tasks with specific priorities and times.
	// p3 created first, p1 created second, p1 created third, p2 created fourth.
	now := time.Now()
	tasks := []struct {
		id       string
		priority int
		created  time.Time
	}{
		{"sort-p3", 3, now.Add(-4 * time.Second)},
		{"sort-p1a", 1, now.Add(-3 * time.Second)},
		{"sort-p1b", 1, now.Add(-2 * time.Second)},
		{"sort-p2", 2, now.Add(-1 * time.Second)},
	}

	for _, tc := range tasks {
		task := &models.Task{
			ID:          tc.id,
			SpaceID:     space.ID,
			CreatedByID: user.ID,
			Title:       "Task " + tc.id,
			Type:        models.TaskTypeTask,
			Status:      models.TaskStatusOpen,
			Priority:    tc.priority,
			CreatedAt:   tc.created,
			UpdatedAt:   tc.created,
		}
		if err := db.Create(task).Error; err != nil {
			t.Fatalf("Failed to create task %s: %v", tc.id, err)
		}
	}

	result, err := svc.GetReadyTasks(space.ID, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(result) != 4 {
		t.Fatalf("Expected 4 tasks, got %d", len(result))
	}

	expectedOrder := []string{"sort-p1a", "sort-p1b", "sort-p2", "sort-p3"}
	for i, expected := range expectedOrder {
		if result[i].ID != expected {
			t.Errorf("Position %d: expected %q, got %q", i, expected, result[i].ID)
		}
	}
}

func TestReadyWorkService_ChainedDependencies(t *testing.T) {
	db := setupReadyWorkTestDB(t)
	svc := NewReadyWorkService(db)

	account := rwTestAccount(t, db)
	user := rwTestUser(t, db, account.ID)
	space := rwTestSpace(t, db, account.ID)

	// Chain: A -> B -> C (C blocked by B, B blocked by A)
	// Only A should be ready.
	rwTestTask(t, db, "chain-a", space.ID, user.ID, models.TaskStatusOpen, 0, nil)
	rwTestTask(t, db, "chain-b", space.ID, user.ID, models.TaskStatusOpen, 0, nil)
	rwTestTask(t, db, "chain-c", space.ID, user.ID, models.TaskStatusOpen, 0, nil)
	rwTestDep(t, db, "chain-b", "chain-a") // B blocked by A
	rwTestDep(t, db, "chain-c", "chain-b") // C blocked by B

	result, err := svc.GetReadyTasks(space.ID, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("Expected 1 ready task, got %d", len(result))
	}
	if result[0].ID != "chain-a" {
		t.Errorf("Expected 'chain-a', got %q", result[0].ID)
	}
}

func TestReadyWorkService_WaitsForDepAlsoBlocks(t *testing.T) {
	db := setupReadyWorkTestDB(t)
	svc := NewReadyWorkService(db)

	account := rwTestAccount(t, db)
	user := rwTestUser(t, db, account.ID)
	space := rwTestSpace(t, db, account.ID)

	// Task B has a "waits_for" dep on A. According to the spec, only "blocks"
	// type deps are considered for blocking. "waits_for" should not block.
	rwTestTask(t, db, "wf-a", space.ID, user.ID, models.TaskStatusOpen, 0, nil)
	rwTestTask(t, db, "wf-b", space.ID, user.ID, models.TaskStatusOpen, 0, nil)

	dep := &models.TaskDep{
		ID:         uuid.New(),
		FromTaskID: "wf-b",
		ToTaskID:   "wf-a",
		Type:       models.DepTypeWaitsFor,
	}
	if err := db.Create(dep).Error; err != nil {
		t.Fatalf("Failed to create dep: %v", err)
	}

	result, err := svc.GetReadyTasks(space.ID, nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("Expected 2 ready tasks, got %d", len(result))
	}
}
