package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/storage"
)

// --- Mock sub-task repo (implements SubTaskRepoInterface) ---

type mockSubTaskRepo struct {
	subtasks map[uuid.UUID]*models.SubTask
	images   map[uuid.UUID]*models.SubTaskImage
}

func newMockSubTaskRepo() *mockSubTaskRepo {
	return &mockSubTaskRepo{
		subtasks: make(map[uuid.UUID]*models.SubTask),
		images:   make(map[uuid.UUID]*models.SubTaskImage),
	}
}

func (m *mockSubTaskRepo) Create(s *models.SubTask) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	m.subtasks[s.ID] = s
	return nil
}

func (m *mockSubTaskRepo) GetByID(id uuid.UUID) (*models.SubTask, error) {
	s, ok := m.subtasks[id]
	if !ok {
		return nil, assertNotFound
	}
	return s, nil
}

func (m *mockSubTaskRepo) ListByTaskID(taskID string) ([]models.SubTask, error) {
	var out []models.SubTask
	for _, s := range m.subtasks {
		if s.TaskID == taskID {
			out = append(out, *s)
		}
	}
	return out, nil
}

func (m *mockSubTaskRepo) GetMaxDisplayOrder(taskID string) (int, error) { return -1, nil }
func (m *mockSubTaskRepo) Update(id uuid.UUID, updates map[string]interface{}) error {
	return nil
}
func (m *mockSubTaskRepo) Delete(id uuid.UUID) error            { delete(m.subtasks, id); return nil }
func (m *mockSubTaskRepo) Reorder(taskID string, ids []uuid.UUID) error { return nil }

func (m *mockSubTaskRepo) AddImage(img *models.SubTaskImage) error {
	if img.ID == uuid.Nil {
		img.ID = uuid.New()
	}
	m.images[img.ID] = img
	return nil
}

func (m *mockSubTaskRepo) GetImageByID(id uuid.UUID) (*models.SubTaskImage, error) {
	img, ok := m.images[id]
	if !ok {
		return nil, assertNotFound
	}
	return img, nil
}

func (m *mockSubTaskRepo) GetMaxImageDisplayOrder(subTaskID uuid.UUID) (int, error) {
	max := -1
	for _, img := range m.images {
		if img.SubTaskID == subTaskID && img.DisplayOrder > max {
			max = img.DisplayOrder
		}
	}
	return max, nil
}

func (m *mockSubTaskRepo) DeleteImage(id uuid.UUID) error {
	if _, ok := m.images[id]; !ok {
		return assertNotFound
	}
	delete(m.images, id)
	return nil
}

// --- helpers ---

func setupSubTaskApp(h *SubTaskHandler, authDTO *dtos.AuthDTO, spaceID uuid.UUID) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("authDTO", authDTO)
		c.Locals("spaceID", spaceID)
		return c.Next()
	})
	scoped := app.Group("/api/v1/spaces/:spaceId")
	h.RegisterSubTaskRoutes(scoped)
	return app
}

func subTaskImageURL(spaceID uuid.UUID, taskID string, subtaskID uuid.UUID, rest string) string {
	return "/api/v1/spaces/" + spaceID.String() + "/tasks/" + taskID + "/subtasks/" + subtaskID.String() + "/images" + rest
}

func TestSubTaskHandler_UploadImage_Success(t *testing.T) {
	store := storage.NewLocalStorage(t.TempDir(), "/uploads", nil)
	repo := newMockSubTaskRepo()
	taskSvc := newMockTaskDepTaskService()

	spaceID := uuid.New()
	taskID := "subimg-task-1"
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}
	st := &models.SubTask{ID: uuid.New(), TaskID: taskID, Title: "Step"}
	repo.subtasks[st.ID] = st

	h := NewSubTaskHandler(repo, taskSvc, store)
	auth := adminAuthDTOWithSpace(uuid.New(), spaceID)
	app := setupSubTaskApp(h, auth, spaceID)

	req := multipartUpload(t, subTaskImageURL(spaceID, taskID, st.ID, ""), "step.png", "image/png", []byte("png"))
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Len(t, repo.images, 1)
}

func TestSubTaskHandler_UploadImage_RequiresAdmin(t *testing.T) {
	store := storage.NewLocalStorage(t.TempDir(), "/uploads", nil)
	repo := newMockSubTaskRepo()
	taskSvc := newMockTaskDepTaskService()

	spaceID := uuid.New()
	taskID := "subimg-task-2"
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}
	st := &models.SubTask{ID: uuid.New(), TaskID: taskID, Title: "Step"}
	repo.subtasks[st.ID] = st

	h := NewSubTaskHandler(repo, taskSvc, store)
	auth := userAuthDTOWithSpace(uuid.New(), spaceID) // non-admin
	app := setupSubTaskApp(h, auth, spaceID)

	req := multipartUpload(t, subTaskImageURL(spaceID, taskID, st.ID, ""), "step.png", "image/png", []byte("png"))
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	assert.Empty(t, repo.images)
}

func TestSubTaskHandler_UploadImage_UnsupportedType(t *testing.T) {
	store := storage.NewLocalStorage(t.TempDir(), "/uploads", []string{"image/png"})
	repo := newMockSubTaskRepo()
	taskSvc := newMockTaskDepTaskService()

	spaceID := uuid.New()
	taskID := "subimg-task-3"
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}
	st := &models.SubTask{ID: uuid.New(), TaskID: taskID, Title: "Step"}
	repo.subtasks[st.ID] = st

	h := NewSubTaskHandler(repo, taskSvc, store)
	auth := adminAuthDTOWithSpace(uuid.New(), spaceID)
	app := setupSubTaskApp(h, auth, spaceID)

	req := multipartUpload(t, subTaskImageURL(spaceID, taskID, st.ID, ""), "x.gif", "image/gif", []byte("gif"))
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
}

func TestSubTaskHandler_DeleteImage(t *testing.T) {
	store := storage.NewLocalStorage(t.TempDir(), "/uploads", nil)
	repo := newMockSubTaskRepo()
	taskSvc := newMockTaskDepTaskService()

	spaceID := uuid.New()
	taskID := "subimg-task-4"
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}
	st := &models.SubTask{ID: uuid.New(), TaskID: taskID, Title: "Step"}
	repo.subtasks[st.ID] = st
	img := &models.SubTaskImage{ID: uuid.New(), SubTaskID: st.ID, ImageURL: "/uploads/sub-task-images/x.png"}
	repo.images[img.ID] = img

	h := NewSubTaskHandler(repo, taskSvc, store)
	auth := adminAuthDTOWithSpace(uuid.New(), spaceID)
	app := setupSubTaskApp(h, auth, spaceID)

	req := httptest.NewRequest(http.MethodDelete, subTaskImageURL(spaceID, taskID, st.ID, "/"+img.ID.String()), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Empty(t, repo.images)
}
