// Package storage provides local-disk persistence for uploaded media files
// (photos and videos attached to tasks and sub-tasks). Files are written under
// a configurable base directory and served back via a public URL path.
package storage

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// DefaultAllowedMimeTypes is used when ALLOWED_MIME_TYPES is not configured.
var DefaultAllowedMimeTypes = []string{
	"image/jpeg", "image/png", "image/webp", "image/gif",
	"video/mp4", "video/quicktime", "video/webm",
}

// extByMime maps allowed MIME types to a canonical file extension. Used when the
// uploaded filename has no usable extension.
var extByMime = map[string]string{
	"image/jpeg":      ".jpg",
	"image/png":       ".png",
	"image/webp":      ".webp",
	"image/gif":       ".gif",
	"video/mp4":       ".mp4",
	"video/quicktime": ".mov",
	"video/webm":      ".webm",
}

// ErrUnsupportedMediaType is returned when an upload's MIME type is not allowed.
type ErrUnsupportedMediaType struct {
	MimeType string
}

func (e *ErrUnsupportedMediaType) Error() string {
	return fmt.Sprintf("unsupported media type: %q", e.MimeType)
}

// SavedFile describes a file that was persisted to disk.
type SavedFile struct {
	// URL is the public path the file is served at (e.g. "/uploads/task-media/<uuid>.jpg").
	URL string
	// FileName is the original client-provided filename.
	FileName string
	// MimeType is the validated content type.
	MimeType string
	// Size is the number of bytes written.
	Size int64
}

// LocalStorage writes uploads to a base directory on the local filesystem and
// serves them under a public URL prefix.
type LocalStorage struct {
	baseDir    string
	publicPath string
	allowed    map[string]bool
}

// NewLocalStorage builds a LocalStorage. baseDir is the on-disk root (e.g.
// "./uploads"), publicPath is the URL prefix files are served under (e.g.
// "/uploads"), and allowedMimeTypes restricts what may be uploaded — if empty,
// DefaultAllowedMimeTypes is used.
func NewLocalStorage(baseDir, publicPath string, allowedMimeTypes []string) *LocalStorage {
	if len(allowedMimeTypes) == 0 {
		allowedMimeTypes = DefaultAllowedMimeTypes
	}
	allowed := make(map[string]bool, len(allowedMimeTypes))
	for _, mt := range allowedMimeTypes {
		mt = strings.TrimSpace(mt)
		if mt != "" {
			allowed[mt] = true
		}
	}
	return &LocalStorage{
		baseDir:    baseDir,
		publicPath: strings.TrimRight(publicPath, "/"),
		allowed:    allowed,
	}
}

// BaseDir returns the on-disk root directory.
func (s *LocalStorage) BaseDir() string { return s.baseDir }

// Allows reports whether the given MIME type may be uploaded.
func (s *LocalStorage) Allows(mimeType string) bool {
	return s.allowed[normalizeMime(mimeType)]
}

// Save validates and writes an uploaded file under baseDir/subdir, returning the
// public URL and metadata. It returns *ErrUnsupportedMediaType if the file's
// MIME type is not allowed.
func (s *LocalStorage) Save(fh *multipart.FileHeader, subdir string) (*SavedFile, error) {
	mimeType := normalizeMime(fh.Header.Get("Content-Type"))
	if !s.allowed[mimeType] {
		return nil, &ErrUnsupportedMediaType{MimeType: mimeType}
	}

	ext := filepath.Ext(fh.Filename)
	if ext == "" {
		ext = extByMime[mimeType]
	}
	name := uuid.NewString() + ext

	dir := filepath.Join(s.baseDir, subdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}

	dst := filepath.Join(dir, name)
	written, err := writeMultipart(fh, dst)
	if err != nil {
		return nil, err
	}

	// URL path always uses forward slashes regardless of OS.
	url := s.publicPath + "/" + path.Join(subdir, name)

	return &SavedFile{
		URL:      url,
		FileName: fh.Filename,
		MimeType: mimeType,
		Size:     written,
	}, nil
}

// Delete removes the file backing the given public URL. A missing file is not
// treated as an error so deletes remain idempotent.
func (s *LocalStorage) Delete(url string) error {
	rel := strings.TrimPrefix(url, s.publicPath+"/")
	if rel == url || rel == "" {
		// URL does not belong to this storage; nothing to delete.
		return nil
	}
	// Guard against path traversal escaping baseDir.
	clean := filepath.Clean(filepath.Join(s.baseDir, filepath.FromSlash(rel)))
	if !strings.HasPrefix(clean, filepath.Clean(s.baseDir)+string(os.PathSeparator)) {
		return fmt.Errorf("refusing to delete outside base dir: %q", url)
	}
	if err := os.Remove(clean); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func writeMultipart(fh *multipart.FileHeader, dst string) (int64, error) {
	src, err := fh.Open()
	if err != nil {
		return 0, fmt.Errorf("open upload: %w", err)
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		return 0, fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	n, err := io.Copy(out, src)
	if err != nil {
		return 0, fmt.Errorf("write file: %w", err)
	}
	return n, nil
}

// normalizeMime strips any parameters (e.g. "; charset=...") and lowercases.
func normalizeMime(mt string) string {
	if i := strings.IndexByte(mt, ';'); i >= 0 {
		mt = mt[:i]
	}
	return strings.ToLower(strings.TrimSpace(mt))
}
