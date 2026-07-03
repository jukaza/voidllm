package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// providerSelectColumns is the ordered column list used in all provider SELECT
// queries. It must match the scan order in scanProvider.
const providerSelectColumns = "id, name, contact_info, status, notes, created_at, updated_at, deleted_at"

// Provider represents an upstream partner supplying API capacity.
type Provider struct {
	ID          string
	Name        string
	ContactInfo string
	Status      string
	Notes       string
	CreatedAt   string
	UpdatedAt   string
	DeletedAt   *string
}

// CreateProviderParams holds the input for creating a provider.
type CreateProviderParams struct {
	Name        string
	ContactInfo string
	Status      string
	Notes       string
}

// UpdateProviderParams holds optional fields for updating a provider.
// A nil pointer means the field is not changed.
type UpdateProviderParams struct {
	Name        *string
	ContactInfo *string
	Status      *string
	Notes       *string
}

// CreateProvider inserts a new provider and returns the persisted record.
func (d *DB) CreateProvider(ctx context.Context, params CreateProviderParams) (*Provider, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("create provider: generate id: %w", err)
	}

	status := params.Status
	if status == "" {
		status = "active"
	}

	p := d.dialect.Placeholder
	insertQuery := "INSERT INTO providers (id, name, contact_info, status, notes, created_at, updated_at) " +
		"VALUES (" + p(1) + ", " + p(2) + ", " + p(3) + ", " + p(4) + ", " + p(5) + ", CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)"
	selectQuery := "SELECT " + providerSelectColumns + " FROM providers WHERE id = " + p(1)

	var provider *Provider
	err = d.WithTx(ctx, func(q Querier) error {
		if _, execErr := q.ExecContext(ctx, insertQuery, id.String(), params.Name, params.ContactInfo, status, params.Notes); execErr != nil {
			return translateError(execErr)
		}
		row := q.QueryRowContext(ctx, selectQuery, id.String())
		var scanErr error
		provider, scanErr = scanProvider(row)
		return scanErr
	})
	if err != nil {
		return nil, fmt.Errorf("create provider: %w", err)
	}
	return provider, nil
}

// GetProvider retrieves an active provider by ID.
// It returns ErrNotFound if the provider does not exist or has been soft-deleted.
func (d *DB) GetProvider(ctx context.Context, id string) (*Provider, error) {
	query := "SELECT " + providerSelectColumns +
		" FROM providers WHERE id = " + d.dialect.Placeholder(1) + " AND deleted_at IS NULL"
	row := d.sql.QueryRowContext(ctx, query, id)
	provider, err := scanProvider(row)
	if err != nil {
		return nil, fmt.Errorf("GetProvider %s: %w", id, translateError(err))
	}
	return provider, nil
}

// ListProviders returns a page of active providers ordered by ID ascending.
// cursor is an exclusive lower bound on ID for keyset pagination; pass "" to
// start from the beginning.
func (d *DB) ListProviders(ctx context.Context, cursor string, limit int) ([]Provider, error) {
	p := d.dialect.Placeholder
	argN := 1
	conditions := []string{"deleted_at IS NULL"}
	var args []any

	if cursor != "" {
		conditions = append(conditions, "id > "+p(argN))
		args = append(args, cursor)
		argN++
	}

	query := "SELECT " + providerSelectColumns + " FROM providers WHERE " +
		strings.Join(conditions, " AND ") + " ORDER BY id ASC LIMIT " + p(argN)
	args = append(args, limit)

	rows, err := d.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ListProviders query: %w", err)
	}
	defer rows.Close()

	var providers []Provider
	for rows.Next() {
		var pr Provider
		if err := rows.Scan(&pr.ID, &pr.Name, &pr.ContactInfo, &pr.Status, &pr.Notes, &pr.CreatedAt, &pr.UpdatedAt, &pr.DeletedAt); err != nil {
			return nil, fmt.Errorf("ListProviders scan: %w", err)
		}
		providers = append(providers, pr)
	}
	return providers, rows.Err()
}

// UpdateProvider applies the non-nil fields in params to an active provider
// and returns the updated record. It returns ErrNotFound if the provider does
// not exist or has been soft-deleted.
func (d *DB) UpdateProvider(ctx context.Context, id string, params UpdateProviderParams) (*Provider, error) {
	p := d.dialect.Placeholder
	var sets []string
	var args []any
	argN := 1

	addSet := func(col string, val any) {
		sets = append(sets, col+" = "+p(argN))
		args = append(args, val)
		argN++
	}

	if params.Name != nil {
		addSet("name", *params.Name)
	}
	if params.ContactInfo != nil {
		addSet("contact_info", *params.ContactInfo)
	}
	if params.Status != nil {
		addSet("status", *params.Status)
	}
	if params.Notes != nil {
		addSet("notes", *params.Notes)
	}

	if len(sets) == 0 {
		return d.GetProvider(ctx, id)
	}

	sets = append(sets, "updated_at = CURRENT_TIMESTAMP")
	query := "UPDATE providers SET " + strings.Join(sets, ", ") +
		" WHERE id = " + p(argN) + " AND deleted_at IS NULL"
	args = append(args, id)

	res, err := d.sql.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("UpdateProvider %s: %w", id, translateError(err))
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("UpdateProvider %s: rows affected: %w", id, err)
	}
	if affected == 0 {
		return nil, fmt.Errorf("UpdateProvider %s: %w", id, ErrNotFound)
	}
	return d.GetProvider(ctx, id)
}

// DeleteProvider soft-deletes a provider. It returns ErrNotFound if the
// provider does not exist or is already deleted.
func (d *DB) DeleteProvider(ctx context.Context, id string) error {
	p := d.dialect.Placeholder
	query := "UPDATE providers SET deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP" +
		" WHERE id = " + p(1) + " AND deleted_at IS NULL"

	res, err := d.sql.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("DeleteProvider %s: %w", id, translateError(err))
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("DeleteProvider %s: rows affected: %w", id, err)
	}
	if affected == 0 {
		return fmt.Errorf("DeleteProvider %s: %w", id, ErrNotFound)
	}
	return nil
}

func scanProvider(row *sql.Row) (*Provider, error) {
	var pr Provider
	err := row.Scan(&pr.ID, &pr.Name, &pr.ContactInfo, &pr.Status, &pr.Notes, &pr.CreatedAt, &pr.UpdatedAt, &pr.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &pr, nil
}
