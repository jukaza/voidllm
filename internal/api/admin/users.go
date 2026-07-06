package admin

import (
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/audit"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/db"
	"golang.org/x/crypto/bcrypt"
)

// createUserRequest is the JSON body accepted by CreateUser.
type createUserRequest struct {
	Email         string `json:"email"`
	DisplayName   string `json:"display_name"`
	Password      string `json:"password"`
	IsSystemAdmin bool   `json:"is_system_admin"`
}

// updateUserRequest is the JSON body accepted by UpdateUser.
type updateUserRequest struct {
	Email         *string `json:"email"`
	DisplayName   *string `json:"display_name"`
	Password      *string `json:"password"`
	IsSystemAdmin *bool   `json:"is_system_admin"`
}

// userResponse is the JSON representation of a user returned by the API.
type userResponse struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	DisplayName   string  `json:"display_name"`
	AuthProvider  string  `json:"auth_provider"`
	IsSystemAdmin bool    `json:"is_system_admin"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
	DeletedAt     *string `json:"deleted_at,omitempty"`
}

// paginatedUsersResponse wraps a page of users with pagination metadata.
type paginatedUsersResponse struct {
	Data    []userResponse `json:"data"`
	HasMore bool           `json:"has_more"`
	Cursor  string         `json:"next_cursor,omitempty"`
}

// userToResponse converts a db.User to its API wire representation.
func userToResponse(u *db.User) userResponse {
	return userResponse{
		ID:            u.ID,
		Email:         u.Email,
		DisplayName:   u.DisplayName,
		AuthProvider:  u.AuthProvider,
		IsSystemAdmin: u.IsSystemAdmin,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
		DeletedAt:     u.DeletedAt,
	}
}

// CreateUser handles POST /api/v1/users.
func (h *Handler) CreateUser(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil {
		return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
	}
	if !auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
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
		Email:         req.Email,
		DisplayName:   req.DisplayName,
		PasswordHash:  &hashStr,
		AuthProvider:  "local",
		IsSystemAdmin: req.IsSystemAdmin,
	})
	if err != nil {
		if errors.Is(err, db.ErrConflict) {
			return apierror.Conflict(c, "email already in use")
		}
		h.Log.ErrorContext(c.Context(), "create user", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to create user")
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
				"email":           req.Email,
				"is_system_admin": strconv.FormatBool(req.IsSystemAdmin),
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
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil {
		return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
	}
	if !auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
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

	return c.JSON(userToResponse(user))
}

// ListUsers handles GET /api/v1/users.
func (h *Handler) ListUsers(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil {
		return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
	}
	if !auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "insufficient permissions")
	}

	p, err := parsePagination(c)
	if err != nil {
		return apierror.BadRequest(c, err.Error())
	}
	includeDeleted := c.Query("include_deleted") == "true" && auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin)

	users, err := h.DB.ListUsers(c.Context(), p.Cursor, p.Limit+1, includeDeleted)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "list users", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to list users")
	}

	hasMore := len(users) > p.Limit
	if hasMore {
		users = users[:p.Limit]
	}

	resp := paginatedUsersResponse{
		Data:    make([]userResponse, len(users)),
		HasMore: hasMore,
	}
	for i := range users {
		resp.Data[i] = userToResponse(&users[i])
	}
	if hasMore && len(users) > 0 {
		resp.Cursor = users[len(users)-1].ID
	}
	return c.JSON(resp)
}

// UpdateUser handles PATCH /api/v1/users/:user_id.
func (h *Handler) UpdateUser(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil {
		return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
	}
	if !auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "insufficient permissions")
	}

	id := c.Params("user_id")

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
		Email:         req.Email,
		DisplayName:   req.DisplayName,
		IsSystemAdmin: req.IsSystemAdmin,
	}

	if req.Password != nil {
		if len(*req.Password) < 8 {
			return apierror.BadRequest(c, "password must be at least 8 characters")
		}
		if len(*req.Password) > 72 {
			return apierror.BadRequest(c, "password must be at most 72 bytes")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			if errors.Is(err, bcrypt.ErrPasswordTooLong) {
				return apierror.BadRequest(c, "password must be at most 72 bytes")
			}
			h.Log.ErrorContext(c.Context(), "update user: bcrypt", slog.String("error", err.Error()))
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
	return c.JSON(userToResponse(user))
}

// DeleteUser handles DELETE /api/v1/users/:user_id.
func (h *Handler) DeleteUser(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil {
		return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "missing authentication")
	}
	if !auth.HasRole(keyInfo.Role, auth.RoleSystemAdmin) {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "insufficient permissions")
	}

	id := c.Params("user_id")

	if err := h.DB.DeleteUser(c.Context(), id); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "user not found")
		}
		h.Log.ErrorContext(c.Context(), "delete user", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to delete user")
	}
	return c.SendStatus(fiber.StatusNoContent)
}
