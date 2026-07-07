package admin

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/audit"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/db"
	"github.com/jukaza/tavo/pkg/keygen"
	"golang.org/x/crypto/bcrypt"
)

type createUserRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	Role        string `json:"role"`
}

type updateUserRequest struct {
	Email       *string `json:"email"`
	DisplayName *string `json:"display_name"`
	Password    *string `json:"password"`
	Role        *string `json:"role"`
	Status      *string `json:"status"`
}

type userResponse struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	DisplayName   string  `json:"display_name"`
	AuthProvider  string  `json:"auth_provider"`
	Role          string  `json:"role"`
	Status        string  `json:"status"`
	IsSystemAdmin bool    `json:"is_system_admin"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
	DeletedAt     *string `json:"deleted_at,omitempty"`
}

type paginatedUsersResponse struct {
	Data    []userResponse `json:"data"`
	HasMore bool           `json:"has_more"`
	Cursor  string         `json:"next_cursor,omitempty"`
}

func userToResponse(u *db.User) userResponse {
	return userResponse{
		ID:            u.ID,
		Email:         u.Email,
		DisplayName:   u.DisplayName,
		AuthProvider:  u.AuthProvider,
		Role:          u.Role,
		Status:        u.Status,
		IsSystemAdmin: u.Role == db.UserRoleRoot || u.Role == db.UserRoleAdmin,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
		DeletedAt:     u.DeletedAt,
	}
}

func actorFromCtx(c fiber.Ctx) (auth.KeyInfo, error) {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil {
		return auth.KeyInfo{}, errors.New("unauthorized")
	}
	return *keyInfo, nil
}

func canManageUser(actorRole string, target *db.User) bool {
	return auth.CanManageRole(actorRole, target.Role)
}

func normalizeAssignableRole(actorRole, requested string) (string, error) {
	role := db.UserRoleMember
	switch requested {
	case "", db.UserRoleMember:
		role = db.UserRoleMember
	case db.UserRoleAdmin:
		if !auth.HasRole(actorRole, auth.RoleRoot) {
			return "", errors.New("only root can assign admin role")
		}
		role = db.UserRoleAdmin
	case db.UserRoleRoot:
		return "", errors.New("cannot assign root role via API")
	default:
		return "", errors.New("invalid role")
	}
	return role, nil
}

// CreateUser handles POST /api/v1/users.
func (h *Handler) CreateUser(c fiber.Ctx) error {
	keyInfo, err := actorFromCtx(c)
	if err != nil {
		return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
	}
	if !auth.HasRole(keyInfo.Role, auth.RoleAdmin) {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "insufficient permissions")
	}

	var req createUserRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}

	req.Email = strings.TrimSpace(req.Email)
	req.DisplayName = strings.TrimSpace(req.DisplayName)

	if req.Email == "" {
		return apierror.BadRequest(c, "email is required")
	}
	if !strings.Contains(req.Email, "@") {
		return apierror.BadRequest(c, "invalid email format")
	}
	if req.DisplayName == "" {
		return apierror.BadRequest(c, "display_name is required")
	}
	if len(req.Password) < 8 {
		return apierror.BadRequest(c, "password must be at least 8 characters")
	}
	if len(req.Password) > 72 {
		return apierror.BadRequest(c, "password must be at most 72 bytes")
	}

	role, err := normalizeAssignableRole(keyInfo.Role, req.Role)
	if err != nil {
		return apierror.BadRequest(c, err.Error())
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		if errors.Is(err, bcrypt.ErrPasswordTooLong) {
			return apierror.BadRequest(c, "password must be at most 72 bytes")
		}
		h.Log.ErrorContext(c.Context(), "create user: bcrypt", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to hash password")
	}
	hashStr := string(hash)

	user, err := h.DB.CreateUser(c.Context(), db.CreateUserParams{
		Email:        req.Email,
		DisplayName:  req.DisplayName,
		PasswordHash: &hashStr,
		AuthProvider: "local",
		Role:         role,
		Status:       db.UserStatusActive,
	})
	if err != nil {
		if errors.Is(err, db.ErrConflict) {
			return apierror.Conflict(c, "email already in use")
		}
		h.Log.ErrorContext(c.Context(), "create user", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to create user")
	}

	if _, err := h.DB.CreateWallet(c.Context(), user.ID, ""); err != nil && !errors.Is(err, db.ErrConflict) {
		h.Log.ErrorContext(c.Context(), "create user wallet", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to create user wallet")
	}

	if h.AuditLogger != nil {
		h.AuditLogger.Log(audit.Event{
			Timestamp:    time.Now().UTC(),
			ActorID:      keyInfo.UserID,
			ActorType:    "user",
			ActorKeyID:   keyInfo.ID,
			Action:       "user_created",
			ResourceType: "user",
			ResourceID:   user.ID,
			Description: marshalDescription(map[string]string{
				"email": req.Email,
				"role":  role,
			}),
			IPAddress:  c.IP(),
			StatusCode: fiber.StatusCreated,
			RequestID:  apierror.RequestIDFromCtx(c),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(userToResponse(user))
}

// GetUser handles GET /api/v1/users/:user_id.
func (h *Handler) GetUser(c fiber.Ctx) error {
	keyInfo, err := actorFromCtx(c)
	if err != nil {
		return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
	}
	if !auth.HasRole(keyInfo.Role, auth.RoleAdmin) {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "insufficient permissions")
	}

	id := c.Params("user_id")
	user, err := h.DB.GetUser(c.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "user not found")
		}
		h.Log.ErrorContext(c.Context(), "get user", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to get user")
	}
	if !canManageUser(keyInfo.Role, user) && user.ID != keyInfo.UserID {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "insufficient permissions")
	}

	return c.JSON(userToResponse(user))
}

// ListUsers handles GET /api/v1/users.
func (h *Handler) ListUsers(c fiber.Ctx) error {
	keyInfo, err := actorFromCtx(c)
	if err != nil {
		return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
	}
	if !auth.HasRole(keyInfo.Role, auth.RoleAdmin) {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "insufficient permissions")
	}

	p, err := parsePagination(c)
	if err != nil {
		return apierror.BadRequest(c, err.Error())
	}
	includeDeleted := c.Query("include_deleted") == "true" && auth.HasRole(keyInfo.Role, auth.RoleRoot)

	users, err := h.DB.ListUsers(c.Context(), db.ListUsersFilter{
		Cursor:         p.Cursor,
		Limit:          p.Limit + 1,
		IncludeDeleted: includeDeleted,
		Search:         c.Query("search"),
		Role:           c.Query("role"),
		Status:         c.Query("status"),
	})
	if err != nil {
		h.Log.ErrorContext(c.Context(), "list users", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to list users")
	}

	hasMore := len(users) > p.Limit
	if hasMore {
		users = users[:p.Limit]
	}

	resp := paginatedUsersResponse{
		Data:    make([]userResponse, 0, len(users)),
		HasMore: hasMore,
	}
	for i := range users {
		if canManageUser(keyInfo.Role, &users[i]) || users[i].ID == keyInfo.UserID {
			resp.Data = append(resp.Data, userToResponse(&users[i]))
		}
	}
	if hasMore && len(users) > 0 {
		resp.Cursor = users[len(users)-1].ID
	}
	return c.JSON(resp)
}

// UpdateUser handles PATCH /api/v1/users/:user_id.
func (h *Handler) UpdateUser(c fiber.Ctx) error {
	keyInfo, err := actorFromCtx(c)
	if err != nil {
		return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
	}
	if !auth.HasRole(keyInfo.Role, auth.RoleAdmin) {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "insufficient permissions")
	}

	id := c.Params("user_id")
	origin, err := h.DB.GetUser(c.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "user not found")
		}
		return apierror.InternalError(c, "failed to get user")
	}
	if !canManageUser(keyInfo.Role, origin) {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "insufficient permissions")
	}

	var req updateUserRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}

	if req.Email != nil {
		trimmed := strings.TrimSpace(*req.Email)
		if trimmed == "" {
			return apierror.BadRequest(c, "email must not be empty")
		}
		if !strings.Contains(trimmed, "@") {
			return apierror.BadRequest(c, "invalid email format")
		}
		req.Email = &trimmed
	}

	if req.DisplayName != nil {
		trimmed := strings.TrimSpace(*req.DisplayName)
		if trimmed == "" {
			return apierror.BadRequest(c, "display_name must not be empty")
		}
		req.DisplayName = &trimmed
	}

	params := db.UpdateUserParams{
		Email:       req.Email,
		DisplayName: req.DisplayName,
	}

	if req.Role != nil {
		role, roleErr := normalizeAssignableRole(keyInfo.Role, *req.Role)
		if roleErr != nil {
			return apierror.BadRequest(c, roleErr.Error())
		}
		if !canManageUser(keyInfo.Role, origin) {
			return apierror.Send(c, fiber.StatusForbidden, "forbidden", "insufficient permissions")
		}
		params.Role = &role
	}

	if req.Status != nil {
		status := strings.TrimSpace(*req.Status)
		if status != db.UserStatusActive && status != db.UserStatusDisabled {
			return apierror.BadRequest(c, "invalid status")
		}
		if status == db.UserStatusDisabled && origin.Role == db.UserRoleRoot {
			return apierror.BadRequest(c, "cannot disable root user")
		}
		params.Status = &status
	}

	if req.Password != nil {
		if len(*req.Password) < 8 {
			return apierror.BadRequest(c, "password must be at least 8 characters")
		}
		if len(*req.Password) > 72 {
			return apierror.BadRequest(c, "password must be at most 72 bytes")
		}
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if hashErr != nil {
			if errors.Is(hashErr, bcrypt.ErrPasswordTooLong) {
				return apierror.BadRequest(c, "password must be at most 72 bytes")
			}
			h.Log.ErrorContext(c.Context(), "update user: bcrypt", slog.String("error", hashErr.Error()))
			return apierror.InternalError(c, "failed to hash password")
		}
		hashStr := string(hash)
		params.PasswordHash = &hashStr
	}

	user, err := h.DB.UpdateUser(c.Context(), id, params)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "user not found")
		}
		if errors.Is(err, db.ErrConflict) {
			return apierror.Conflict(c, "email already in use")
		}
		h.Log.ErrorContext(c.Context(), "update user", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to update user")
	}

	if req.Status != nil && *req.Status == db.UserStatusDisabled {
		_ = h.revokeAllUserSessions(c, id)
	}

	return c.JSON(userToResponse(user))
}

// DeleteUser handles DELETE /api/v1/users/:user_id.
func (h *Handler) DeleteUser(c fiber.Ctx) error {
	keyInfo, err := actorFromCtx(c)
	if err != nil {
		return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
	}
	if !auth.HasRole(keyInfo.Role, auth.RoleAdmin) {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "insufficient permissions")
	}

	id := c.Params("user_id")
	if id == keyInfo.UserID {
		return apierror.BadRequest(c, "cannot delete your own account")
	}

	origin, err := h.DB.GetUser(c.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "user not found")
		}
		return apierror.InternalError(c, "failed to get user")
	}
	if !canManageUser(keyInfo.Role, origin) {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "insufficient permissions")
	}
	if origin.Role == db.UserRoleRoot {
		return apierror.BadRequest(c, "cannot delete root user")
	}

	if err := h.DB.DeleteUser(c.Context(), id); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "user not found")
		}
		h.Log.ErrorContext(c.Context(), "delete user", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to delete user")
	}
	_ = h.revokeAllUserSessions(c, id)
	return c.SendStatus(fiber.StatusNoContent)
}

// ManageUser handles POST /api/v1/users/:user_id/manage for enable/disable shortcuts.
func (h *Handler) ManageUser(c fiber.Ctx) error {
	keyInfo, err := actorFromCtx(c)
	if err != nil {
		return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
	}
	if !auth.HasRole(keyInfo.Role, auth.RoleAdmin) {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "insufficient permissions")
	}

	id := c.Params("user_id")
	var req struct {
		Action string `json:"action"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}

	origin, err := h.DB.GetUser(c.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "user not found")
		}
		return apierror.InternalError(c, "failed to get user")
	}
	if !canManageUser(keyInfo.Role, origin) {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "insufficient permissions")
	}

	var status string
	switch req.Action {
	case "enable":
		status = db.UserStatusActive
	case "disable":
		if origin.Role == db.UserRoleRoot {
			return apierror.BadRequest(c, "cannot disable root user")
		}
		status = db.UserStatusDisabled
	default:
		return apierror.BadRequest(c, "invalid action")
	}

	user, err := h.DB.UpdateUser(c.Context(), id, db.UpdateUserParams{Status: &status})
	if err != nil {
		return apierror.InternalError(c, "failed to update user")
	}
	if status == db.UserStatusDisabled {
		_ = h.revokeAllUserSessions(c, id)
	}

	if h.AuditLogger != nil {
		h.AuditLogger.Log(audit.Event{
			Timestamp:    time.Now().UTC(),
			ActorID:      keyInfo.UserID,
			ActorType:    "user",
			ActorKeyID:   keyInfo.ID,
			Action:       "user_" + req.Action,
			ResourceType: "user",
			ResourceID:   id,
			Description:  marshalDescription(map[string]string{"action": req.Action}),
			IPAddress:    c.IP(),
			StatusCode:   fiber.StatusOK,
			RequestID:    apierror.RequestIDFromCtx(c),
		})
	}

	return c.JSON(userToResponse(user))
}

func (h *Handler) revokeAllUserSessions(c fiber.Ctx, userID string) error {
	if err := h.DB.RevokeUserSessions(c.Context(), userID); err != nil {
		return err
	}
	var toEvict []string
	h.KeyCache.Range(func(keyHash string, ki auth.KeyInfo) bool {
		if ki.UserID == userID && ki.KeyType == keygen.KeyTypeSession {
			toEvict = append(toEvict, keyHash)
		}
		return true
	})
	for _, keyHash := range toEvict {
		h.KeyCache.Delete(keyHash)
	}
	return nil
}