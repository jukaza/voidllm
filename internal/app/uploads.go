package app

import (
	"io"
	"log/slog"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/jukaza/tavo/internal/site"
)

// registerSiteUploads serves uploaded site assets (custom logos) from disk.
// Must be registered before the SPA catch-all.
func registerSiteUploads(app *fiber.App, dataDir string, log *slog.Logger) {
	dir := site.LogoDir(dataDir)
	app.Get("/uploads/site/*", func(c fiber.Ctx) error {
		name := strings.TrimPrefix(c.Params("*"), "/")
		if name == "" || strings.Contains(name, "..") || strings.Contains(name, "/") {
			return fiber.ErrNotFound
		}
		if !strings.HasPrefix(name, "logo.") {
			return fiber.ErrNotFound
		}

		path := filepath.Join(dir, name)
		f, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				return fiber.ErrNotFound
			}
			log.ErrorContext(c.Context(), "uploads: open file", slog.String("error", err.Error()))
			return fiber.ErrInternalServerError
		}
		defer f.Close()

		stat, err := f.Stat()
		if err != nil || stat.IsDir() {
			return fiber.ErrNotFound
		}

		data, err := io.ReadAll(f)
		if err != nil {
			return fiber.ErrInternalServerError
		}

		ext := filepath.Ext(name)
		ct := mime.TypeByExtension(ext)
		if ct == "" {
			ct = "application/octet-stream"
		}
		c.Set("Content-Type", ct)
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("Cache-Control", "public, max-age=300")
		return c.Send(data)
	})
}