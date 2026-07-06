package admin

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/auth"
	"github.com/voidmind-io/voidllm/internal/backup"
)

func (h *Handler) requireBackup(c fiber.Ctx) (*backup.Service, error) {
	if h.Backup == nil {
		return nil, apierror.InternalError(c, "backup service unavailable")
	}
	return h.Backup, nil
}

func (h *Handler) GetBackupS3Config(c fiber.Ctx) error {
	svc, err := h.requireBackup(c)
	if err != nil {
		return err
	}
	cfg, err := svc.GetS3Config(c.Context())
	if err != nil {
		h.Log.ErrorContext(c.Context(), "backup: get s3 config", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load backup storage config")
	}
	return c.JSON(cfg)
}

func (h *Handler) UpdateBackupS3Config(c fiber.Ctx) error {
	svc, err := h.requireBackup(c)
	if err != nil {
		return err
	}
	var input backup.S3Config
	if err := c.Bind().JSON(&input); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	cfg, err := svc.UpdateS3Config(c.Context(), input)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "backup: update s3 config", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to save backup storage config")
	}
	return c.JSON(cfg)
}

func (h *Handler) TestBackupS3Config(c fiber.Ctx) error {
	svc, err := h.requireBackup(c)
	if err != nil {
		return err
	}
	var input backup.S3Config
	if err := c.Bind().JSON(&input); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if err := svc.TestS3Connection(c.Context(), input); err != nil {
		return c.JSON(fiber.Map{"ok": false, "message": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true, "message": "connection successful"})
}

func (h *Handler) GetBackupSchedule(c fiber.Ctx) error {
	svc, err := h.requireBackup(c)
	if err != nil {
		return err
	}
	cfg, err := svc.GetSchedule(c.Context())
	if err != nil {
		return apierror.InternalError(c, "failed to load backup schedule")
	}
	slots, _ := svc.SlotPreview(cfg)
	return c.JSON(fiber.Map{
		"schedule": cfg,
		"slots":    slots,
	})
}

func (h *Handler) UpdateBackupSchedule(c fiber.Ctx) error {
	svc, err := h.requireBackup(c)
	if err != nil {
		return err
	}
	var input backup.ScheduleConfig
	if err := c.Bind().JSON(&input); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	cfg, err := svc.UpdateSchedule(c.Context(), input)
	if err != nil {
		if strings.Contains(err.Error(), "backups_per_day") || errors.Is(err, backup.ErrInvalidType) {
			return apierror.BadRequest(c, err.Error())
		}
		return apierror.InternalError(c, "failed to save backup schedule")
	}
	slots, _ := svc.SlotPreview(cfg)
	return c.JSON(fiber.Map{
		"schedule": cfg,
		"slots":    slots,
	})
}

func (h *Handler) CreateBackup(c fiber.Ctx) error {
	svc, err := h.requireBackup(c)
	if err != nil {
		return err
	}
	var req backup.CreateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if req.BackupType == "" {
		req.BackupType = backup.TypeConfig
	}
	rec, err := svc.StartBackup(c.Context(), req.BackupType)
	if err != nil {
		if errors.Is(err, backup.ErrS3NotConfigured) {
			return apierror.BadRequest(c, err.Error())
		}
		if errors.Is(err, backup.ErrInProgress) {
			return apierror.Conflict(c, err.Error())
		}
		if errors.Is(err, backup.ErrInvalidType) {
			return apierror.BadRequest(c, err.Error())
		}
		return apierror.InternalError(c, "failed to start backup")
	}
	return c.Status(fiber.StatusAccepted).JSON(rec)
}

func (h *Handler) ListBackups(c fiber.Ctx) error {
	svc, err := h.requireBackup(c)
	if err != nil {
		return err
	}
	items, err := svc.ListBackups(c.Context())
	if err != nil {
		return apierror.InternalError(c, "failed to list backups")
	}
	if items == nil {
		items = []backup.Record{}
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *Handler) GetBackup(c fiber.Ctx) error {
	svc, err := h.requireBackup(c)
	if err != nil {
		return err
	}
	rec, err := svc.GetBackup(c.Context(), c.Params("id"))
	if err != nil {
		if errors.Is(err, backup.ErrNotFound) {
			return apierror.NotFound(c, "backup not found")
		}
		return apierror.InternalError(c, "failed to load backup")
	}
	return c.JSON(rec)
}

func (h *Handler) DeleteBackup(c fiber.Ctx) error {
	svc, err := h.requireBackup(c)
	if err != nil {
		return err
	}
	if err := svc.DeleteBackup(c.Context(), c.Params("id")); err != nil {
		if errors.Is(err, backup.ErrNotFound) {
			return apierror.NotFound(c, "backup not found")
		}
		if errors.Is(err, backup.ErrS3NotConfigured) {
			return apierror.BadRequest(c, err.Error())
		}
		return apierror.InternalError(c, "failed to delete backup")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) GetBackupDownloadURL(c fiber.Ctx) error {
	svc, err := h.requireBackup(c)
	if err != nil {
		return err
	}
	url, err := svc.DownloadURL(c.Context(), c.Params("id"))
	if err != nil {
		if errors.Is(err, backup.ErrNotFound) {
			return apierror.NotFound(c, "backup not found")
		}
		if errors.Is(err, backup.ErrNotCompleted) {
			return apierror.BadRequest(c, err.Error())
		}
		return apierror.InternalError(c, "failed to create download URL")
	}
	return c.JSON(fiber.Map{"url": url})
}

func (h *Handler) RestoreBackup(c fiber.Ctx) error {
	svc, err := h.requireBackup(c)
	if err != nil {
		return err
	}
	var req backup.RestoreRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if strings.TrimSpace(req.Password) == "" {
		return apierror.BadRequest(c, "password is required")
	}
	preBackup := true
	if req.PreBackup != nil {
		preBackup = *req.PreBackup
	}
	ki := auth.KeyInfoFromCtx(c)
	if ki == nil || ki.UserID == "" {
		return apierror.Unauthorized(c, "authentication required")
	}
	if err := svc.StartRestore(c.Context(), c.Params("id"), ki.UserID, req.Password, preBackup); err != nil {
		if errors.Is(err, backup.ErrNotFound) {
			return apierror.NotFound(c, "backup not found")
		}
		if errors.Is(err, backup.ErrNotCompleted) {
			return apierror.BadRequest(c, err.Error())
		}
		if errors.Is(err, backup.ErrRestoreRunning) || errors.Is(err, backup.ErrInProgress) {
			return apierror.Conflict(c, err.Error())
		}
		if strings.Contains(err.Error(), "invalid password") {
			return apierror.Unauthorized(c, "invalid password")
		}
		return apierror.InternalError(c, "failed to start restore")
	}
	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"status": "restore_started"})
}