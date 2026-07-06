package admin

import (
	"bufio"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/usage"
)

// UsageStream handles GET /api/v1/usage/stream — SSE live usage counters.
func (h *Handler) UsageStream(c fiber.Ctx) error {
	if auth.KeyInfoFromCtx(c) == nil {
		return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
	}
	if h.LiveStats == nil {
		return apierror.Send(c, fiber.StatusServiceUnavailable, "unavailable", "live stats not enabled")
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	stats := h.LiveStats
	return c.SendStreamWriter(func(w *bufio.Writer) {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		write := func() {
			snap := stats.Snapshot()
			payload, err := json.Marshal(snap)
			if err != nil {
				return
			}
			_, _ = w.WriteString("data: ")
			_, _ = w.Write(payload)
			_, _ = w.WriteString("\n\n")
			_ = w.Flush()
		}

		write()
		for {
			select {
			case <-c.Context().Done():
				return
			case <-ticker.C:
				write()
			}
		}
	})
}

// UsageLive handles GET /api/v1/usage/live — one-shot live stats snapshot.
func (h *Handler) UsageLive(c fiber.Ctx) error {
	if auth.KeyInfoFromCtx(c) == nil {
		return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
	}
	if h.LiveStats == nil {
		return c.JSON(usage.LiveSnapshot{})
	}
	return c.JSON(h.LiveStats.Snapshot())
}