package subscription

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

// CoverUploadURLPrefix is the public URL path for uploaded package covers.
const CoverUploadURLPrefix = "/uploads/subscriptions/"

const maxCoverBytes = 4 << 20 // 4 MiB

var allowedCoverMIME = map[string]string{
	"image/png":     ".png",
	"image/jpeg":    ".jpg",
	"image/webp":    ".webp",
	"image/svg+xml": ".svg",
	"image/svg":     ".svg",
}

// DefaultCoverPresets are built-in gradient ids for package covers.
var DefaultCoverPresets = []string{"aurora", "sunset", "ocean", "ember", "violet"}

// CoverDir returns the on-disk directory for uploaded subscription covers.
func CoverDir(dataDir string) string {
	return filepath.Join(dataDir, "uploads", "subscriptions")
}

// IsUploadedCover reports whether coverValue is served from local uploads.
func IsUploadedCover(coverValue string) bool {
	return strings.HasPrefix(strings.TrimSpace(coverValue), CoverUploadURLPrefix)
}

// RemovePackageCover deletes uploaded cover files for packageID.
func RemovePackageCover(dataDir, packageID string) error {
	dir := CoverDir(dataDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	prefix := packageID + "."
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// SavePackageCover stores a validated cover file and returns its public URL path.
func SavePackageCover(dataDir, packageID string, header *multipart.FileHeader) (string, error) {
	if header == nil {
		return "", fmt.Errorf("cover file is required")
	}
	if header.Size <= 0 {
		return "", fmt.Errorf("cover file is empty")
	}
	if header.Size > maxCoverBytes {
		return "", fmt.Errorf("cover file exceeds 4 MB limit")
	}

	ext, err := coverExtension(header)
	if err != nil {
		return "", err
	}

	dir := CoverDir(dataDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create cover dir: %w", err)
	}

	_ = RemovePackageCover(dataDir, packageID)

	filename := packageID + ext
	dest := filepath.Join(dir, filename)

	src, err := header.Open()
	if err != nil {
		return "", fmt.Errorf("open cover: %w", err)
	}
	defer src.Close()

	out, err := os.Create(dest)
	if err != nil {
		return "", fmt.Errorf("create cover file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, io.LimitReader(src, maxCoverBytes+1)); err != nil {
		_ = os.Remove(dest)
		return "", fmt.Errorf("write cover: %w", err)
	}

	return CoverUploadURLPrefix + filename, nil
}

func coverExtension(header *multipart.FileHeader) (string, error) {
	contentType := strings.ToLower(strings.TrimSpace(header.Header.Get("Content-Type")))
	ext, ok := allowedCoverMIME[contentType]
	if !ok || ext == "" {
		nameExt := strings.ToLower(filepath.Ext(header.Filename))
		switch nameExt {
		case ".png":
			ext = ".png"
		case ".jpg", ".jpeg":
			ext = ".jpg"
		case ".webp":
			ext = ".webp"
		case ".svg":
			ext = ".svg"
		default:
			return "", fmt.Errorf("unsupported cover image type")
		}
	}
	return ext, nil
}

// ValidCoverType reports whether t is a known cover_type value.
func ValidCoverType(t string) bool {
	switch t {
	case "upload", "default", "url":
		return true
	default:
		return false
	}
}

// ValidQuotaPolicy reports whether p is a known quota_exceeded_policy value.
func ValidQuotaPolicy(p string) bool {
	switch p {
	case "block", "fallback_wallet":
		return true
	default:
		return false
	}
}