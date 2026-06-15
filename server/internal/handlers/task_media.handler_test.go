package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tylerjvollick/nori/internal/dtos"
	"github.com/tylerjvollick/nori/internal/models"
	"github.com/tylerjvollick/nori/internal/storage"
)

// --- Mock task media repo ---

type mockTaskMediaRepo struct {
	media     map[uuid.UUID]*models.TaskMedia
	createErr error
}

func newMockTaskMediaRepo() *mockTaskMediaRepo {
	return &mockTaskMediaRepo{media: make(map[uuid.UUID]*models.TaskMedia)}
}

func (m *mockTaskMediaRepo) Create(media *models.TaskMedia) error {
	if m.createErr != nil {
		return m.createErr
	}
	if media.ID == uuid.Nil {
		media.ID = uuid.New()
	}
	m.media[media.ID] = media
	return nil
}

func (m *mockTaskMediaRepo) GetByID(id uuid.UUID) (*models.TaskMedia, error) {
	med, ok := m.media[id]
	if !ok {
		return nil, assertNotFound
	}
	return med, nil
}

func (m *mockTaskMediaRepo) ListByTaskID(taskID string) ([]models.TaskMedia, error) {
	var out []models.TaskMedia
	for _, med := range m.media {
		if med.TaskID == taskID {
			out = append(out, *med)
		}
	}
	return out, nil
}

func (m *mockTaskMediaRepo) GetMaxDisplayOrder(taskID string) (int, error) {
	max := -1
	for _, med := range m.media {
		if med.TaskID == taskID && med.DisplayOrder > max {
			max = med.DisplayOrder
		}
	}
	return max, nil
}

func (m *mockTaskMediaRepo) Delete(id uuid.UUID) error {
	if _, ok := m.media[id]; !ok {
		return assertNotFound
	}
	delete(m.media, id)
	return nil
}

var assertNotFound = &notFoundError{}

type notFoundError struct{}

func (e *notFoundError) Error() string { return "not found" }

// --- Test helpers ---

func setupTaskMediaApp(h *TaskMediaHandler, authDTO *dtos.AuthDTO, spaceID uuid.UUID) *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("authDTO", authDTO)
		c.Locals("spaceID", spaceID)
		return c.Next()
	})
	scoped := app.Group("/api/v1/spaces/:spaceId")
	h.RegisterTaskMediaRoutes(scoped)
	return app
}

func mediaURL(spaceID uuid.UUID, taskID, rest string) string {
	return "/api/v1/spaces/" + spaceID.String() + "/tasks/" + taskID + "/media" + rest
}

// multipartUpload builds a multipart request body with a single "file" field.
func multipartUpload(t *testing.T, url, filename, contentType string, body []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	hdr := make(map[string][]string)
	hdr["Content-Disposition"] = []string{`form-data; name="file"; filename="` + filename + `"`}
	hdr["Content-Type"] = []string{contentType}
	part, err := w.CreatePart(hdr)
	require.NoError(t, err)
	_, err = part.Write(body)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, url, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestTaskMediaHandler_Upload_Success(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewLocalStorage(dir, "/uploads", nil)
	repo := newMockTaskMediaRepo()
	taskSvc := newMockTaskDepTaskService()

	spaceID := uuid.New()
	accountID := uuid.New()
	taskID := "task-upload-1"
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}

	h := NewTaskMediaHandler(repo, taskSvc, store)
	auth := userAuthDTOWithSpace(accountID, spaceID)
	app := setupTaskMediaApp(h, auth, spaceID)

	req := multipartUpload(t, mediaURL(spaceID, taskID, ""), "photo.jpg", "image/jpeg", []byte("jpeg-bytes"))
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var got dtos.TaskMediaResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
	assert.Equal(t, taskID, got.TaskID)
	assert.True(t, strings.HasPrefix(got.URL, "/uploads/task-media/"), "url: %s", got.URL)
	assert.Equal(t, "photo.jpg", got.FileName)
	assert.Equal(t, "image/jpeg", got.MimeType)
	assert.NotNil(t, got.CapturedByID)

	// File written to disk.
	rel := strings.TrimPrefix(got.URL, "/uploads/")
	_, statErr := os.Stat(filepath.Join(dir, rel))
	assert.NoError(t, statErr)

	// Record persisted.
	assert.Len(t, repo.media, 1)
}

func TestTaskMediaHandler_Upload_UnsupportedType(t *testing.T) {
	store := storage.NewLocalStorage(t.TempDir(), "/uploads", []string{"image/png"})
	repo := newMockTaskMediaRepo()
	taskSvc := newMockTaskDepTaskService()

	spaceID := uuid.New()
	taskID := "task-bad-mime"
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}

	h := NewTaskMediaHandler(repo, taskSvc, store)
	auth := userAuthDTOWithSpace(uuid.New(), spaceID)
	app := setupTaskMediaApp(h, auth, spaceID)

	req := multipartUpload(t, mediaURL(spaceID, taskID, ""), "bad.exe", "application/octet-stream", []byte("MZ"))
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
	assert.Empty(t, repo.media)
}

func TestTaskMediaHandler_Upload_MissingFile(t *testing.T) {
	store := storage.NewLocalStorage(t.TempDir(), "/uploads", nil)
	repo := newMockTaskMediaRepo()
	taskSvc := newMockTaskDepTaskService()

	spaceID := uuid.New()
	taskID := "task-no-file"
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}

	h := NewTaskMediaHandler(repo, taskSvc, store)
	auth := userAuthDTOWithSpace(uuid.New(), spaceID)
	app := setupTaskMediaApp(h, auth, spaceID)

	req := httptest.NewRequest(http.MethodPost, mediaURL(spaceID, taskID, ""), nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestTaskMediaHandler_Upload_TaskInOtherSpace(t *testing.T) {
	store := storage.NewLocalStorage(t.TempDir(), "/uploads", nil)
	repo := newMockTaskMediaRepo()
	taskSvc := newMockTaskDepTaskService()

	spaceID := uuid.New()
	otherSpace := uuid.New()
	taskID := "task-other-space"
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: otherSpace}

	h := NewTaskMediaHandler(repo, taskSvc, store)
	auth := userAuthDTOWithSpace(uuid.New(), spaceID)
	app := setupTaskMediaApp(h, auth, spaceID)

	req := multipartUpload(t, mediaURL(spaceID, taskID, ""), "photo.jpg", "image/jpeg", []byte("x"))
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestTaskMediaHandler_ListAndDelete(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewLocalStorage(dir, "/uploads", nil)
	repo := newMockTaskMediaRepo()
	taskSvc := newMockTaskDepTaskService()

	spaceID := uuid.New()
	taskID := "task-list-del"
	taskSvc.tasks[taskID] = &models.Task{ID: taskID, SpaceID: spaceID}

	// Seed one media record + backing file.
	saved, err := store.Save(fileHeaderFromBytes(t, "a.jpg", "image/jpeg", []byte("a")), "task-media")
	require.NoError(t, err)
	med := &models.TaskMedia{ID: uuid.New(), TaskID: taskID, FilePath: saved.URL, FileName: "a.jpg", MimeType: "image/jpeg"}
	repo.media[med.ID] = med

	h := NewTaskMediaHandler(repo, taskSvc, store)
	auth := userAuthDTOWithSpace(uuid.New(), spaceID)
	app := setupTaskMediaApp(h, auth, spaceID)

	// List.
	listReq := httptest.NewRequest(http.MethodGet, mediaURL(spaceID, taskID, ""), nil)
	listResp, err := app.Test(listReq, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, listResp.StatusCode)
	var list dtos.TaskMediaListResponse
	require.NoError(t, json.NewDecoder(listResp.Body).Decode(&list))
	assert.Equal(t, 1, list.Total)

	// Delete.
	delReq := httptest.NewRequest(http.MethodDelete, mediaURL(spaceID, taskID, "/"+med.ID.String()), nil)
	delResp, err := app.Test(delReq, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, delResp.StatusCode)
	assert.Empty(t, repo.media)

	// Backing file removed.
	rel := strings.TrimPrefix(saved.URL, "/uploads/")
	_, statErr := os.Stat(filepath.Join(dir, rel))
	assert.True(t, os.IsNotExist(statErr))
}

// fileHeaderFromBytes builds a *multipart.FileHeader for direct storage tests.
func fileHeaderFromBytes(t *testing.T, filename, contentType string, body []byte) *multipart.FileHeader {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	hdr := make(map[string][]string)
	hdr["Content-Disposition"] = []string{`form-data; name="file"; filename="` + filename + `"`}
	hdr["Content-Type"] = []string{contentType}
	part, err := w.CreatePart(hdr)
	require.NoError(t, err)
	_, err = part.Write(body)
	require.NoError(t, err)
	require.NoError(t, w.Close())

	r := multipart.NewReader(&buf, w.Boundary())
	form, err := r.ReadForm(1 << 20)
	require.NoError(t, err)
	return form.File["file"][0]
}
