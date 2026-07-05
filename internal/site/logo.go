package site

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

// LogoUploadURLPrefix is the public URL path for uploaded site logos.
const LogoUploadURLPrefix = "/uploads/site/"

const maxLogoBytes = 2 << 20 // 2 MiB

var allowedLogoMIME = map[string]string{
	"image/png":                ".png",
	"image/jpeg":               ".jpg",
	"image/webp":               ".webp",
	"image/svg+xml":            ".svg",
	"image/svg":                ".svg",
	"application/octet-stream": "", // resolved from filename when needed
}

// LogoDir returns the on-disk directory for uploaded site logos.
func LogoDir(dataDir string) string {
	return filepath.Join(dataDir, "uploads", "site")
}

// IsUploadedLogo reports whether logo is served from the uploads directory.
func IsUploadedLogo(logoPath string) bool {
	return strings.HasPrefix(strings.TrimSpace(logoPath), LogoUploadURLPrefix)
}

// RemoveUploadedLogo deletes custom logo files when resetting to the bundled default.
func RemoveUploadedLogo(dataDir string) error {
	dir := LogoDir(dataDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "logo.") {
			continue
		}
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// SaveUploadedLogo stores a validated logo file and returns its public URL path.
func SaveUploadedLogo(dataDir string, header *multipart.FileHeader) (string, error) {
	if header == nil {
		return "", fmt.Errorf("logo file is required")
	}
	if header.Size <= 0 {
		return "", fmt.Errorf("logo file is empty")
	}
	if header.Size > maxLogoBytes {
		return "", fmt.Errorf("logo file exceeds 2 MB limit")
	}

	ext, err := logoExtension(header)
	if err != nil {
		return "", err
	}

	dir := LogoDir(dataDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create logo dir: %w", err)
	}
	if err := RemoveUploadedLogo(dataDir); err != nil {
		return "", fmt.Errorf("remove previous logo: %w", err)
	}

	src, err := header.Open()
	if err != nil {
		return "", fmt.Errorf("open logo: %w", err)
	}
	defer src.Close()

	filename := "logo" + ext
	dstPath := filepath.Join(dir, filename)
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return "", fmt.Errorf("create logo file: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, io.LimitReader(src, maxLogoBytes+1))
	if err != nil {
		_ = os.Remove(dstPath)
		return "", fmt.Errorf("write logo: %w", err)
	}
	if written > maxLogoBytes {
		_ = os.Remove(dstPath)
		return "", fmt.Errorf("logo file exceeds 2 MB limit")
	}

	return LogoUploadURLPrefix + filename, nil
}

func logoExtension(header *multipart.FileHeader) (string, error) {
	contentType := strings.ToLower(strings.TrimSpace(header.Header.Get("Content-Type")))
	if ext, ok := allowedLogoMIME[contentType]; ok && ext != "" {
		return ext, nil
	}
	if ext, ok := allowedLogoMIME[contentType]; ok && ext == "" {
		ext = strings.ToLower(filepath.Ext(header.Filename))
		switch ext {
		case ".png", ".jpg", ".jpeg", ".webp", ".svg":
			if ext == ".jpeg" {
				return ".jpg", nil
			}
			return ext, nil
		default:
			return "", fmt.Errorf("unsupported logo file type")
		}
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	switch ext {
	case ".png":
		return ".png", nil
	case ".jpg", ".jpeg":
		return ".jpg", nil
	case ".webp":
		return ".webp", nil
	case ".svg":
		return ".svg", nil
	default:
		return "", fmt.Errorf("logo must be PNG, JPEG, WebP, or SVG")
	}
}