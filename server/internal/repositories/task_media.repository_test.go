package repositories

import (
	"testing"

	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

// cleanMediaTables connects to the test DB and clears task/media tables in
// FK-safe order so each test starts from a clean slate. setupTestDB only clears
// account/space/user tables and fails to clear users while task rows still
// reference them, so we delete the task-related tables first then re-clear.
func cleanMediaTables(t *testing.T) *gorm.DB {
	d := setupTestDB(t)
	for _, tbl := range []string{
		"task_media", "sub_task_images", "sub_tasks", "task_dep", "task",
		"space_member", "spaces", "users", "accounts",
	} {
		d.Exec("DELETE FROM " + tbl)
	}
	return d
}

func TestTaskMediaRepository_CreateAndList(t *testing.T) {
	d := cleanMediaTables(t)

	account := createTestAccount(t, d)
	user := createTestUser(t, d, "tmedia-create@example.com", account.ID)
	space := createTestSpace(t, d, "TMedia Create Space", account.ID)
	task := createTestTask(t, d, "tmedia-create-1", space.ID, user.ID)

	repo := NewTaskMediaRepository(d)

	m := &models.TaskMedia{
		TaskID:       task.ID,
		FilePath:     "/uploads/task-media/abc.jpg",
		FileName:     "abc.jpg",
		MimeType:     "image/jpeg",
		FileSize:     1234,
		DisplayOrder: 0,
		CapturedByID: &user.ID,
	}
	if err := repo.Create(m); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if m.ID.String() == "" {
		t.Fatal("expected ID to be populated after Create")
	}

	media, err := repo.ListByTaskID(task.ID)
	if err != nil {
		t.Fatalf("ListByTaskID failed: %v", err)
	}
	if len(media) != 1 {
		t.Fatalf("expected 1 media, got %d", len(media))
	}
	if media[0].FilePath != "/uploads/task-media/abc.jpg" {
		t.Errorf("unexpected FilePath %q", media[0].FilePath)
	}
	if media[0].CapturedByID == nil || *media[0].CapturedByID != user.ID {
		t.Errorf("expected CapturedByID %v, got %v", user.ID, media[0].CapturedByID)
	}
}

func TestTaskMediaRepository_GetMaxDisplayOrder(t *testing.T) {
	d := cleanMediaTables(t)

	account := createTestAccount(t, d)
	user := createTestUser(t, d, "tmedia-order@example.com", account.ID)
	space := createTestSpace(t, d, "TMedia Order Space", account.ID)
	task := createTestTask(t, d, "tmedia-order-1", space.ID, user.ID)

	repo := NewTaskMediaRepository(d)

	// No media yet → -1.
	max, err := repo.GetMaxDisplayOrder(task.ID)
	if err != nil {
		t.Fatalf("GetMaxDisplayOrder failed: %v", err)
	}
	if max != -1 {
		t.Errorf("expected -1 for empty, got %d", max)
	}

	for i := 0; i < 3; i++ {
		m := &models.TaskMedia{
			TaskID:       task.ID,
			FilePath:     "/uploads/task-media/x.jpg",
			FileName:     "x.jpg",
			MimeType:     "image/jpeg",
			DisplayOrder: i,
		}
		if err := repo.Create(m); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	max, err = repo.GetMaxDisplayOrder(task.ID)
	if err != nil {
		t.Fatalf("GetMaxDisplayOrder failed: %v", err)
	}
	if max != 2 {
		t.Errorf("expected max 2, got %d", max)
	}
}

func TestTaskMediaRepository_GetByIDAndDelete(t *testing.T) {
	d := cleanMediaTables(t)

	account := createTestAccount(t, d)
	user := createTestUser(t, d, "tmedia-del@example.com", account.ID)
	space := createTestSpace(t, d, "TMedia Del Space", account.ID)
	task := createTestTask(t, d, "tmedia-del-1", space.ID, user.ID)

	repo := NewTaskMediaRepository(d)

	m := &models.TaskMedia{
		TaskID:   task.ID,
		FilePath: "/uploads/task-media/del.jpg",
		FileName: "del.jpg",
		MimeType: "image/jpeg",
	}
	if err := repo.Create(m); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.GetByID(m.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.ID != m.ID {
		t.Errorf("expected ID %v, got %v", m.ID, got.ID)
	}

	if err := repo.Delete(m.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, err := repo.GetByID(m.ID); err == nil {
		t.Error("expected error fetching deleted media")
	}

	// Deleting a non-existent record returns an error.
	if err := repo.Delete(m.ID); err == nil {
		t.Error("expected error deleting already-deleted media")
	}
}
