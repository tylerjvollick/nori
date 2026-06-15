package repositories

import (
	"testing"

	"github.com/tylerjvollick/nori/internal/models"
)

// createTestSubTask inserts a sub-task for image tests.
func createTestSubTask(t *testing.T, repo *SubTaskRepository, taskID, title string) *models.SubTask {
	t.Helper()
	st := &models.SubTask{TaskID: taskID, Title: title}
	if err := repo.Create(st); err != nil {
		t.Fatalf("create sub-task: %v", err)
	}
	return st
}

func TestSubTaskRepository_ImageLifecycle(t *testing.T) {
	d := cleanMediaTables(t)

	account := createTestAccount(t, d)
	user := createTestUser(t, d, "subimg@example.com", account.ID)
	space := createTestSpace(t, d, "SubImg Space", account.ID)
	task := createTestTask(t, d, "subimg-1", space.ID, user.ID)

	repo := NewSubTaskRepository(d)
	st := createTestSubTask(t, repo, task.ID, "Cut the board")

	// No images yet → max display order is -1.
	if max, err := repo.GetMaxImageDisplayOrder(st.ID); err != nil || max != -1 {
		t.Fatalf("expected (-1, nil), got (%d, %v)", max, err)
	}

	img1 := &models.SubTaskImage{SubTaskID: st.ID, ImageURL: "/uploads/sub-task-images/a.jpg", DisplayOrder: 0}
	img2 := &models.SubTaskImage{SubTaskID: st.ID, ImageURL: "/uploads/sub-task-images/b.jpg", DisplayOrder: 1}
	if err := repo.AddImage(img1); err != nil {
		t.Fatalf("AddImage img1: %v", err)
	}
	if err := repo.AddImage(img2); err != nil {
		t.Fatalf("AddImage img2: %v", err)
	}

	if max, err := repo.GetMaxImageDisplayOrder(st.ID); err != nil || max != 1 {
		t.Fatalf("expected (1, nil), got (%d, %v)", max, err)
	}

	// Preload: GetByID and ListByTaskID must return the images ordered.
	got, err := repo.GetByID(st.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(got.Images) != 2 {
		t.Fatalf("expected 2 preloaded images, got %d", len(got.Images))
	}
	if got.Images[0].ImageURL != "/uploads/sub-task-images/a.jpg" {
		t.Errorf("expected images ordered by display_order, got %q first", got.Images[0].ImageURL)
	}

	listed, err := repo.ListByTaskID(task.ID)
	if err != nil {
		t.Fatalf("ListByTaskID: %v", err)
	}
	if len(listed) != 1 || len(listed[0].Images) != 2 {
		t.Fatalf("expected 1 sub-task with 2 images, got %d sub-tasks", len(listed))
	}

	// GetImageByID returns the image.
	fetched, err := repo.GetImageByID(img1.ID)
	if err != nil {
		t.Fatalf("GetImageByID: %v", err)
	}
	if fetched.SubTaskID != st.ID {
		t.Errorf("expected SubTaskID %v, got %v", st.ID, fetched.SubTaskID)
	}

	// DeleteImage removes it.
	if err := repo.DeleteImage(img1.ID); err != nil {
		t.Fatalf("DeleteImage: %v", err)
	}
	if _, err := repo.GetImageByID(img1.ID); err == nil {
		t.Error("expected error fetching deleted image")
	}
	if err := repo.DeleteImage(img1.ID); err == nil {
		t.Error("expected error deleting already-deleted image")
	}

	// Deleting the sub-task cascades to remaining images (FK ON DELETE CASCADE).
	if err := repo.Delete(st.ID); err != nil {
		t.Fatalf("Delete sub-task: %v", err)
	}
	if _, err := repo.GetImageByID(img2.ID); err == nil {
		t.Error("expected image to be cascade-deleted with sub-task")
	}
}
