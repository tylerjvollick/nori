package storage

import (
	"bytes"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeFileHeader builds an in-memory *multipart.FileHeader with the given
// filename, content type, and body — mirroring what Fiber produces from a real
// multipart upload.
func makeFileHeader(t *testing.T, filename, contentType string, body []byte) *multipart.FileHeader {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	if contentType != "" {
		h.Set("Content-Type", contentType)
	}
	part, err := w.CreatePart(h)
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	if _, err := part.Write(body); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	r := multipart.NewReader(&buf, w.Boundary())
	form, err := r.ReadForm(1 << 20)
	if err != nil {
		t.Fatalf("read form: %v", err)
	}
	files := form.File["file"]
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	return files[0]
}

func TestLocalStorage_Save_WritesFileAndReturnsURL(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir, "/uploads", nil)

	fh := makeFileHeader(t, "photo.jpg", "image/jpeg", []byte("fake-jpeg-bytes"))
	saved, err := s.Save(fh, "task-media")
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if !strings.HasPrefix(saved.URL, "/uploads/task-media/") {
		t.Errorf("unexpected URL %q", saved.URL)
	}
	if !strings.HasSuffix(saved.URL, ".jpg") {
		t.Errorf("expected .jpg extension, got %q", saved.URL)
	}
	if saved.MimeType != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %q", saved.MimeType)
	}
	if saved.FileName != "photo.jpg" {
		t.Errorf("expected original filename preserved, got %q", saved.FileName)
	}
	if saved.Size != int64(len("fake-jpeg-bytes")) {
		t.Errorf("expected size %d, got %d", len("fake-jpeg-bytes"), saved.Size)
	}

	// The file must actually exist on disk under the base dir.
	rel := strings.TrimPrefix(saved.URL, "/uploads/")
	onDisk := filepath.Join(dir, rel)
	if _, err := os.Stat(onDisk); err != nil {
		t.Fatalf("expected file on disk at %q: %v", onDisk, err)
	}
}

func TestLocalStorage_Save_RejectsDisallowedMime(t *testing.T) {
	s := NewLocalStorage(t.TempDir(), "/uploads", []string{"image/png"})

	fh := makeFileHeader(t, "evil.exe", "application/octet-stream", []byte("MZ"))
	_, err := s.Save(fh, "task-media")
	if err == nil {
		t.Fatal("expected error for disallowed mime type")
	}
	var unsupported *ErrUnsupportedMediaType
	if !asUnsupported(err, &unsupported) {
		t.Fatalf("expected ErrUnsupportedMediaType, got %T: %v", err, err)
	}
}

func TestLocalStorage_Save_DerivesExtensionFromMime(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir, "/uploads", nil)

	// No extension on the filename — extension should come from the MIME type.
	fh := makeFileHeader(t, "noext", "image/png", []byte("png"))
	saved, err := s.Save(fh, "sub-task-images")
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if !strings.HasSuffix(saved.URL, ".png") {
		t.Errorf("expected .png extension derived from mime, got %q", saved.URL)
	}
}

func TestLocalStorage_Delete_RemovesFileIdempotently(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir, "/uploads", nil)

	fh := makeFileHeader(t, "photo.webp", "image/webp", []byte("webp"))
	saved, err := s.Save(fh, "task-media")
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if err := s.Delete(saved.URL); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	// Deleting again must not error (idempotent).
	if err := s.Delete(saved.URL); err != nil {
		t.Fatalf("second Delete returned error: %v", err)
	}

	rel := strings.TrimPrefix(saved.URL, "/uploads/")
	if _, err := os.Stat(filepath.Join(dir, rel)); !os.IsNotExist(err) {
		t.Errorf("expected file removed, stat err = %v", err)
	}
}

func TestLocalStorage_Delete_RefusesTraversal(t *testing.T) {
	dir := t.TempDir()
	s := NewLocalStorage(dir, "/uploads", nil)

	// A crafted URL trying to escape the base dir must be refused.
	err := s.Delete("/uploads/../../etc/passwd")
	if err == nil {
		t.Fatal("expected error refusing path traversal")
	}
}

func TestLocalStorage_Allows(t *testing.T) {
	s := NewLocalStorage(t.TempDir(), "/uploads", []string{"image/jpeg", "video/mp4"})
	if !s.Allows("image/jpeg") {
		t.Error("expected image/jpeg allowed")
	}
	if !s.Allows("image/jpeg; charset=binary") {
		t.Error("expected mime with params to be normalized and allowed")
	}
	if s.Allows("application/pdf") {
		t.Error("expected application/pdf disallowed")
	}
}

// asUnsupported is a tiny errors.As wrapper to avoid importing errors in the
// test body for a single use.
func asUnsupported(err error, target **ErrUnsupportedMediaType) bool {
	for err != nil {
		if e, ok := err.(*ErrUnsupportedMediaType); ok {
			*target = e
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
