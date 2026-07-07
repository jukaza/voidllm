package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const (
	UserRoleMember = "member"
	UserRoleAdmin  = "admin"
	UserRoleRoot   = "root"

	UserStatusActive   = "active"
	UserStatusDisabled = "disabled"
)

// userSelectColumns is the ordered column list used in all user SELECT queries.
const userSelectColumns = "id, email, display_name, auth_provider, role, status, " +
	"created_at, updated_at, deleted_at"

// User represents a user record in the database.
type User struct {
	ID           string
	Email        string
	DisplayName  string
	AuthProvider string
	Role         string
	Status       string
	CreatedAt    string
	UpdatedAt    string
	DeletedAt    *string
}

// CreateUserParams holds the input for creating a user.
type CreateUserParams struct {
	Email        string
	DisplayName  string
	PasswordHash *string
	AuthProvider string
	ExternalID   *string
	Role         string
	Status       string
}

// UpdateUserParams holds optional fields for updating a user.
type UpdateUserParams struct {
	Email        *string
	DisplayName  *string
	PasswordHash *string
	Role         *string
	Status       *string
}

// ListUsersFilter holds optional list/search filters.
type ListUsersFilter struct {
	Cursor         string
	Limit          int
	IncludeDeleted bool
	Search         string
	Role           string
	Status         string
}

func normalizeUserRole(role string) string {
	switch role {
	case UserRoleAdmin, UserRoleRoot:
		return role
	default:
		return UserRoleMember
	}
}

func normalizeUserStatus(status string) string {
	if status == UserStatusDisabled {
		return UserStatusDisabled
	}
	return UserStatusActive
}

// CreateUser inserts a new user and returns the persisted record.
func (d *DB) CreateUser(ctx context.Context, params CreateUserParams) (*User, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("create user: generate id: %w", err)
	}

	authProvider := params.AuthProvider
	if authProvider == "" {
		authProvider = "local"
	}
	role := normalizeUserRole(params.Role)
	status := normalizeUserStatus(params.Status)

	p := d.dialect.Placeholder
	insertQuery := "INSERT INTO users " +
		"(id, email, display_name, password_hash, auth_provider, external_id, role, status, created_at, updated_at) " +
		"VALUES (" +
		p(1) + ", " + p(2) + ", " + p(3) + ", " + p(4) + ", " +
		p(5) + ", " + p(6) + ", " + p(7) + ", " + p(8) + ", " +
		"CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)"

	selectQuery := "SELECT " + userSelectColumns +
		" FROM users WHERE id = " + p(1) + " AND deleted_at IS NULL"

	var user *User
	err = d.WithTx(ctx, func(q Querier) error {
		_, execErr := q.ExecContext(ctx, insertQuery,
			id.String(),
			params.Email,
			params.DisplayName,
			params.PasswordHash,
			authProvider,
			params.ExternalID,
			role,
			status,
		)
		if execErr != nil {
			return translateError(execErr)
		}

		row := q.QueryRowContext(ctx, selectQuery, id.String())
		var scanErr error
		user, scanErr = scanUser(row)
		return scanErr
	})
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

// CreateUserWithMembership is a compatibility wrapper around CreateUser.
func (d *DB) CreateUserWithMembership(ctx context.Context, userParams CreateUserParams, orgID, role string) (*User, error) {
	return d.CreateUser(ctx, userParams)
}

// GetUser retrieves an active user by their ID.
func (d *DB) GetUser(ctx context.Context, id string) (*User, error) {
	query := "SELECT " + userSelectColumns +
		" FROM users WHERE id = " + d.dialect.Placeholder(1) + " AND deleted_at IS NULL"

	row := d.sql.QueryRowContext(ctx, query, id)
	user, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("GetUser %s: %w", id, translateError(err))
	}
	return user, nil
}

// GetUserByExternalID retrieves an active user by provider + external ID.
func (d *DB) GetUserByExternalID(ctx context.Context, provider, externalID string) (*User, error) {
	query := "SELECT " + userSelectColumns +
		" FROM users WHERE auth_provider = " + d.dialect.Placeholder(1) +
		" AND external_id = " + d.dialect.Placeholder(2) +
		" AND deleted_at IS NULL"

	row := d.sql.QueryRowContext(ctx, query, provider, externalID)
	user, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("GetUserByExternalID: %w", translateError(err))
	}
	return user, nil
}

// GetUserByEmail retrieves an active user by their email address.
func (d *DB) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := "SELECT " + userSelectColumns +
		" FROM users WHERE email = " + d.dialect.Placeholder(1) + " AND deleted_at IS NULL"

	row := d.sql.QueryRowContext(ctx, query, email)
	user, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("GetUserByEmail: %w", translateError(err))
	}
	return user, nil
}

// ListUsers returns a page of users with optional search/filter.
func (d *DB) ListUsers(ctx context.Context, filter ListUsersFilter) ([]User, error) {
	p := d.dialect.Placeholder
	argN := 1
	var conditions []string
	var args []any

	if !filter.IncludeDeleted {
		conditions = append(conditions, "deleted_at IS NULL")
	}
	if filter.Cursor != "" {
		conditions = append(conditions, "id > "+p(argN))
		args = append(args, filter.Cursor)
		argN++
	}
	if filter.Role != "" {
		conditions = append(conditions, "role = "+p(argN))
		args = append(args, filter.Role)
		argN++
	}
	if filter.Status != "" {
		conditions = append(conditions, "status = "+p(argN))
		args = append(args, filter.Status)
		argN++
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + search + "%"
		conditions = append(conditions, "(email LIKE "+p(argN)+" OR display_name LIKE "+p(argN+1)+" OR id = "+p(argN+2)+")")
		args = append(args, like, like, search)
		argN += 3
	}

	query := "SELECT " + userSelectColumns + " FROM users"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY id ASC LIMIT " + p(argN)
	args = append(args, filter.Limit)

	rows, err := d.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ListUsers query: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		u, err := scanUserFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("ListUsers scan: %w", err)
		}
		users = append(users, *u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListUsers rows: %w", err)
	}

	return users, nil
}

// UpdateUser applies a partial update to an active user.
func (d *DB) UpdateUser(ctx context.Context, id string, params UpdateUserParams) (*User, error) {
	p := d.dialect.Placeholder
	argN := 1
	var setClauses []string
	var args []any

	if params.Email != nil {
		setClauses = append(setClauses, "email = "+p(argN))
		args = append(args, *params.Email)
		argN++
	}
	if params.DisplayName != nil {
		setClauses = append(setClauses, "display_name = "+p(argN))
		args = append(args, *params.DisplayName)
		argN++
	}
	if params.PasswordHash != nil {
		setClauses = append(setClauses, "password_hash = "+p(argN))
		args = append(args, *params.PasswordHash)
		argN++
	}
	if params.Role != nil {
		setClauses = append(setClauses, "role = "+p(argN))
		args = append(args, normalizeUserRole(*params.Role))
		argN++
	}
	if params.Status != nil {
		setClauses = append(setClauses, "status = "+p(argN))
		args = append(args, normalizeUserStatus(*params.Status))
		argN++
	}

	if len(setClauses) == 0 {
		return d.GetUser(ctx, id)
	}

	setClauses = append(setClauses, "updated_at = CURRENT_TIMESTAMP")

	updateQuery := "UPDATE users SET " + strings.Join(setClauses, ", ") +
		" WHERE id = " + p(argN) + " AND deleted_at IS NULL"
	args = append(args, id)

	selectQuery := "SELECT " + userSelectColumns +
		" FROM users WHERE id = " + p(1) + " AND deleted_at IS NULL"

	var user *User
	err := d.WithTx(ctx, func(q Querier) error {
		result, execErr := q.ExecContext(ctx, updateQuery, args...)
		if execErr != nil {
			return translateError(execErr)
		}

		n, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			return fmt.Errorf("rows affected: %w", rowsErr)
		}
		if n == 0 {
			return ErrNotFound
		}

		row := q.QueryRowContext(ctx, selectQuery, id)
		var scanErr error
		user, scanErr = scanUser(row)
		return scanErr
	})
	if err != nil {
		return nil, fmt.Errorf("UpdateUser %s: %w", id, err)
	}
	return user, nil
}

// DeleteUser soft-deletes an active user by setting deleted_at.
func (d *DB) DeleteUser(ctx context.Context, id string) error {
	p := d.dialect.Placeholder
	query := "UPDATE users SET deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP " +
		"WHERE id = " + p(1) + " AND deleted_at IS NULL"

	result, err := d.sql.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("DeleteUser %s: %w", id, translateError(err))
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("DeleteUser %s rows affected: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("DeleteUser %s: %w", id, ErrNotFound)
	}

	return nil
}

// GetUserPasswordHash retrieves user ID and bcrypt hash for login.
func (d *DB) GetUserPasswordHash(ctx context.Context, email string) (string, string, error) {
	query := "SELECT id, password_hash FROM users WHERE email = " +
		d.dialect.Placeholder(1) + " AND deleted_at IS NULL AND status = 'active'"

	var id string
	var passwordHash *string
	err := d.sql.QueryRowContext(ctx, query, email).Scan(&id, &passwordHash)
	if err != nil {
		return "", "", fmt.Errorf("GetUserPasswordHash: %w", translateError(err))
	}
	if passwordHash == nil {
		return "", "", ErrNoPassword
	}
	return id, *passwordHash, nil
}

// GetUserPasswordHashByID retrieves auth_provider and bcrypt hash by user ID.
func (d *DB) GetUserPasswordHashByID(ctx context.Context, userID string) (authProvider string, hash string, err error) {
	query := "SELECT auth_provider, password_hash FROM users WHERE id = " +
		d.dialect.Placeholder(1) + " AND deleted_at IS NULL"

	var passwordHash *string
	scanErr := d.sql.QueryRowContext(ctx, query, userID).Scan(&authProvider, &passwordHash)
	if scanErr != nil {
		return "", "", fmt.Errorf("GetUserPasswordHashByID: %w", translateError(scanErr))
	}
	if passwordHash == nil {
		return authProvider, "", ErrNoPassword
	}
	return authProvider, *passwordHash, nil
}

// GetUserAuthState returns role and status for auth checks.
func (d *DB) GetUserAuthState(ctx context.Context, userID string) (role string, status string, err error) {
	p := d.dialect.Placeholder
	err = d.sql.QueryRowContext(ctx,
		"SELECT role, status FROM users WHERE id = "+p(1)+" AND deleted_at IS NULL",
		userID,
	).Scan(&role, &status)
	if err != nil {
		return "", "", fmt.Errorf("GetUserAuthState: %w", translateError(err))
	}
	return role, status, nil
}

func (d *DB) ResolveUserRole(ctx context.Context, userID string) (role string, orgID string, err error) {
	role, status, err := d.GetUserAuthState(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("ResolveUserRole: %w", err)
	}
	if status == UserStatusDisabled {
		return "", "", ErrNotFound
	}
	return role, "", nil
}

func scanUser(row *sql.Row) (*User, error) {
	var u User
	err := row.Scan(
		&u.ID, &u.Email, &u.DisplayName, &u.AuthProvider,
		&u.Role, &u.Status,
		&u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func scanUserFromRows(rows *sql.Rows) (*User, error) {
	var u User
	err := rows.Scan(
		&u.ID, &u.Email, &u.DisplayName, &u.AuthProvider,
		&u.Role, &u.Status,
		&u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}