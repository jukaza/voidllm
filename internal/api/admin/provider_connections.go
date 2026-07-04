package admin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/db"
	"github.com/voidmind-io/voidllm/pkg/crypto"
)

func connectionAAD(id string) []byte {
	return []byte("provider_connection:" + id)
}

type createConnectionRequest struct {
	Name     string `json:"name"`
	APIKey   string `json:"api_key"`
	AuthType string `json:"auth_type"`
	Priority int    `json:"priority"`
	// Bulk: newline-separated "name|key" lines (optional).
	Bulk string `json:"bulk"`
}

type updateConnectionRequest struct {
	Name     *string `json:"name"`
	APIKey   *string `json:"api_key"`
	Priority *int    `json:"priority"`
	IsActive *bool   `json:"is_active"`
}

type reorderConnectionsRequest struct {
	OrderedIDs []string `json:"ordered_ids"`
}

func connectionToJSON(c *db.ProviderConnection) fiber.Map {
	locks := c.ModelLocks
	if locks == nil {
		locks = map[string]string{}
	}
	var earliestLock *string
	now := time.Now().UTC()
	for _, until := range locks {
		t, err := time.Parse(time.RFC3339, until)
		if err != nil || !t.After(now) {
			continue
		}
		s := until
		if earliestLock == nil || t.Before(mustParseRFC3339(*earliestLock)) {
			earliestLock = &s
		}
	}
	if c.LockedUntil != nil {
		if t, err := time.Parse(time.RFC3339, *c.LockedUntil); err == nil && t.After(now) {
			if earliestLock == nil || t.Before(mustParseRFC3339(*earliestLock)) {
				earliestLock = c.LockedUntil
			}
		}
	}

	return fiber.Map{
		"id":                    c.ID,
		"provider_id":           c.ProviderID,
		"name":                  c.Name,
		"auth_type":             c.AuthType,
		"priority":              c.Priority,
		"is_active":             c.IsActive,
		"has_api_key":           c.APIKeyEncrypted != nil,
		"test_status":           c.TestStatus,
		"last_error":            c.LastError,
		"error_code":            c.ErrorCode,
		"last_error_at":         c.LastErrorAt,
		"backoff_level":         c.BackoffLevel,
		"locked_until":          c.LockedUntil,
		"model_locks":           locks,
		"earliest_lock_until":   earliestLock,
		"last_used_at":          c.LastUsedAt,
		"consecutive_use_count": c.ConsecutiveUseCount,
		"created_at":            c.CreatedAt,
		"updated_at":            c.UpdatedAt,
	}
}

func mustParseRFC3339(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func (h *Handler) ensureProvider(ctx context.Context, providerID string) (*db.Provider, error) {
	return h.DB.GetProvider(ctx, providerID)
}

// ListProviderConnections handles GET /api/v1/providers/:provider_id/connections
func (h *Handler) ListProviderConnections(c fiber.Ctx) error {
	ctx := c.Context()
	providerID := c.Params("provider_id")
	if _, err := h.ensureProvider(ctx, providerID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "provider not found")
		}
		return apierror.InternalError(c, "failed to load provider")
	}

	activeOnly := c.Query("active_only") == "1" || strings.EqualFold(c.Query("active_only"), "true")
	conns, err := h.DB.ListProviderConnections(ctx, providerID, activeOnly)
	if err != nil {
		h.Log.ErrorContext(ctx, "list provider connections", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to list connections")
	}

	data := make([]fiber.Map, 0, len(conns))
	for i := range conns {
		data = append(data, connectionToJSON(&conns[i]))
	}
	return c.JSON(fiber.Map{"data": data})
}

// CreateProviderConnection handles POST /api/v1/providers/:provider_id/connections
func (h *Handler) CreateProviderConnection(c fiber.Ctx) error {
	ctx := c.Context()
	providerID := c.Params("provider_id")
	if _, err := h.ensureProvider(ctx, providerID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "provider not found")
		}
		return apierror.InternalError(c, "failed to load provider")
	}

	var req createConnectionRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}

	type result struct {
		Name    string `json:"name"`
		ID      string `json:"id,omitempty"`
		Error   string `json:"error,omitempty"`
	}
	var results []result

	if strings.TrimSpace(req.Bulk) != "" {
		lines := strings.Split(req.Bulk, "\n")
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "|", 2)
			name := strings.TrimSpace(parts[0])
			apiKey := ""
			if len(parts) >= 2 {
				apiKey = strings.TrimSpace(parts[1])
			} else {
				apiKey = name
				name = "Key"
			}
			if name == "" {
				name = "Key"
			}
			if len(lines) > 1 {
				name = fmt.Sprintf("%s %d", name, i+1)
			}
			conn, err := h.createOneConnection(ctx, providerID, name, apiKey, req.AuthType, 0)
			if err != nil {
				results = append(results, result{Name: name, Error: err.Error()})
				continue
			}
			results = append(results, result{Name: conn.Name, ID: conn.ID})
		}
		if len(results) == 0 {
			return apierror.BadRequest(c, "bulk is empty")
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"results": results})
	}

	if strings.TrimSpace(req.Name) == "" {
		return apierror.BadRequest(c, "name is required")
	}
	conn, err := h.createOneConnection(ctx, providerID, strings.TrimSpace(req.Name), strings.TrimSpace(req.APIKey), req.AuthType, req.Priority)
	if err != nil {
		h.Log.ErrorContext(ctx, "create provider connection", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to create connection")
	}
	return c.Status(fiber.StatusCreated).JSON(connectionToJSON(conn))
}

func (h *Handler) createOneConnection(ctx context.Context, providerID, name, apiKey, authType string, priority int) (*db.ProviderConnection, error) {
	var enc *string
	if apiKey != "" {
		// Encrypt with a temporary AAD based on provider until we have connection id.
		// Re-encrypt after create with connection-specific AAD.
		tmp, err := crypto.EncryptString(apiKey, h.EncryptionKey, providerAAD(providerID))
		if err != nil {
			return nil, err
		}
		enc = &tmp
	}
	conn, err := h.DB.CreateProviderConnection(ctx, db.CreateProviderConnectionParams{
		ProviderID: providerID, Name: name, AuthType: authType, Priority: priority,
		APIKeyEncrypted: enc,
	})
	if err != nil {
		return nil, err
	}
	if apiKey != "" {
		reenc, err := crypto.EncryptString(apiKey, h.EncryptionKey, connectionAAD(conn.ID))
		if err != nil {
			_ = h.DB.DeleteProviderConnection(ctx, conn.ID)
			return nil, fmt.Errorf("re-encrypt connection key: %w", err)
		}
		updated, updErr := h.DB.UpdateProviderConnection(ctx, conn.ID, db.UpdateProviderConnectionParams{
			APIKeyEncrypted: &reenc,
		})
		if updErr != nil {
			_ = h.DB.DeleteProviderConnection(ctx, conn.ID)
			return nil, updErr
		}
		return updated, nil
	}
	return conn, nil
}

// UpdateProviderConnection handles PATCH /api/v1/providers/:provider_id/connections/:connection_id
func (h *Handler) UpdateProviderConnection(c fiber.Ctx) error {
	ctx := c.Context()
	providerID := c.Params("provider_id")
	connectionID := c.Params("connection_id")

	conn, err := h.DB.GetProviderConnection(ctx, connectionID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "connection not found")
		}
		return apierror.InternalError(c, "failed to load connection")
	}
	if conn.ProviderID != providerID {
		return apierror.NotFound(c, "connection not found")
	}

	var req updateConnectionRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}

	params := db.UpdateProviderConnectionParams{
		Name:     req.Name,
		Priority: req.Priority,
		IsActive: req.IsActive,
	}
	if req.APIKey != nil {
		if *req.APIKey == "" {
			params.ClearAPIKey = true
		} else {
			enc, encErr := crypto.EncryptString(*req.APIKey, h.EncryptionKey, connectionAAD(connectionID))
			if encErr != nil {
				return apierror.InternalError(c, "failed to store api key")
			}
			params.APIKeyEncrypted = &enc
		}
	}

	updated, err := h.DB.UpdateProviderConnection(ctx, connectionID, params)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "connection not found")
		}
		h.Log.ErrorContext(ctx, "update provider connection", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to update connection")
	}
	return c.JSON(connectionToJSON(updated))
}

// UnlockProviderConnection handles POST /api/v1/providers/:provider_id/connections/:connection_id/unlock
func (h *Handler) UnlockProviderConnection(c fiber.Ctx) error {
	ctx := c.Context()
	providerID := c.Params("provider_id")
	connectionID := c.Params("connection_id")

	conn, err := h.DB.GetProviderConnection(ctx, connectionID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "connection not found")
		}
		return apierror.InternalError(c, "failed to load connection")
	}
	if conn.ProviderID != providerID {
		return apierror.NotFound(c, "connection not found")
	}

	updated, err := h.DB.ClearProviderConnectionLocks(ctx, connectionID)
	if err != nil {
		h.Log.ErrorContext(ctx, "unlock provider connection", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to unlock connection")
	}
	return c.JSON(connectionToJSON(updated))
}

// DeleteProviderConnection handles DELETE /api/v1/providers/:provider_id/connections/:connection_id
func (h *Handler) DeleteProviderConnection(c fiber.Ctx) error {
	ctx := c.Context()
	providerID := c.Params("provider_id")
	connectionID := c.Params("connection_id")

	conn, err := h.DB.GetProviderConnection(ctx, connectionID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "connection not found")
		}
		return apierror.InternalError(c, "failed to load connection")
	}
	if conn.ProviderID != providerID {
		return apierror.NotFound(c, "connection not found")
	}

	if err := h.DB.DeleteProviderConnection(ctx, connectionID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "connection not found")
		}
		return apierror.InternalError(c, "failed to delete connection")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ReorderProviderConnections handles POST /api/v1/providers/:provider_id/connections/reorder
func (h *Handler) ReorderProviderConnections(c fiber.Ctx) error {
	ctx := c.Context()
	providerID := c.Params("provider_id")
	if _, err := h.ensureProvider(ctx, providerID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "provider not found")
		}
		return apierror.InternalError(c, "failed to load provider")
	}

	var req reorderConnectionsRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if len(req.OrderedIDs) == 0 {
		return apierror.BadRequest(c, "ordered_ids is required")
	}

	if err := h.DB.ReorderProviderConnections(ctx, providerID, req.OrderedIDs); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.BadRequest(c, "unknown connection id in ordered_ids")
		}
		h.Log.ErrorContext(ctx, "reorder provider connections", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to reorder connections")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// TestProviderConnection handles POST /api/v1/providers/:provider_id/connections/:connection_id/test
func (h *Handler) TestProviderConnection(c fiber.Ctx) error {
	ctx := c.Context()
	providerID := c.Params("provider_id")
	connectionID := c.Params("connection_id")

	prov, err := h.ensureProvider(ctx, providerID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "provider not found")
		}
		return apierror.InternalError(c, "failed to load provider")
	}
	conn, err := h.DB.GetProviderConnection(ctx, connectionID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "connection not found")
		}
		return apierror.InternalError(c, "failed to load connection")
	}
	if conn.ProviderID != providerID {
		return apierror.NotFound(c, "connection not found")
	}

	apiKey := ""
	if conn.APIKeyEncrypted != nil {
		apiKey, err = crypto.DecryptString(*conn.APIKeyEncrypted, h.EncryptionKey, connectionAAD(conn.ID))
		if err != nil {
			return apierror.InternalError(c, "failed to decrypt api key")
		}
	}
	baseURL := strings.TrimRight(prov.BaseURL, "/")
	if baseURL == "" {
		return apierror.BadRequest(c, "provider base_url is not configured")
	}

	testURL := baseURL + "/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testURL, nil)
	if err != nil {
		return apierror.InternalError(c, "failed to build test request")
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		msg := err.Error()
		if len(msg) > 200 {
			msg = msg[:200]
		}
		now := time.Now().UTC().Format(time.RFC3339)
		unavailable := "unavailable"
		code := 0
		_, _ = h.DB.UpdateProviderConnection(ctx, conn.ID, db.UpdateProviderConnectionParams{
			TestStatus: &unavailable, LastError: &msg, ErrorCode: &code, LastErrorAt: &now,
		})
		return c.JSON(fiber.Map{"ok": false, "error": msg})
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	status := resp.StatusCode
	now := time.Now().UTC().Format(time.RFC3339)
	if status >= 200 && status < 300 {
		active := "active"
		empty := ""
		zero := 0
		updated, _ := h.DB.UpdateProviderConnection(ctx, conn.ID, db.UpdateProviderConnectionParams{
			TestStatus: &active, LastError: &empty, ErrorCode: &zero, LastErrorAt: &now,
			ClearErrorCode: true, BackoffLevel: &zero,
		})
		out := fiber.Map{"ok": true, "status": status}
		if updated != nil {
			out["connection"] = connectionToJSON(updated)
		}
		return c.JSON(out)
	}

	errMsg := fmt.Sprintf("upstream returned %d", status)
	unavailable := "unavailable"
	updated, _ := h.DB.UpdateProviderConnection(ctx, conn.ID, db.UpdateProviderConnectionParams{
		TestStatus: &unavailable, LastError: &errMsg, ErrorCode: &status, LastErrorAt: &now,
	})
	out := fiber.Map{"ok": false, "status": status, "error": errMsg}
	if updated != nil {
		out["connection"] = connectionToJSON(updated)
	}
	return c.JSON(out)
}