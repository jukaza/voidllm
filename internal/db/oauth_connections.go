package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// OAuthConnection is a linked external identity for a user.
type OAuthConnection struct {
	ID         string
	UserID     string
	Provider   string
	ExternalID string
	Label      string
	CreatedAt  string
}

// ListOAuthConnections returns all OAuth links for a user.
func (d *DB) ListOAuthConnections(ctx context.Context, userID string) ([]OAuthConnection, error) {
	p := d.dialect.Placeholder
	query := "SELECT id, user_id, provider, external_id, COALESCE(label, ''), created_at " +
		"FROM user_oauth_connections WHERE user_id = " + p(1) + " ORDER BY provider"
	rows, err := d.SQL().QueryContext(ctx, query, userID)
	if err != nil {
		return nil, translateError(err)
	}
	defer rows.Close()

	var out []OAuthConnection
	for rows.Next() {
		var c OAuthConnection
		if err := rows.Scan(&c.ID, &c.UserID, &c.Provider, &c.ExternalID, &c.Label, &c.CreatedAt); err != nil {
			return nil, translateError(err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetOAuthConnectionByExternal finds a link by provider + external subject id.
func (d *DB) GetOAuthConnectionByExternal(ctx context.Context, provider, externalID string) (*OAuthConnection, error) {
	p := d.dialect.Placeholder
	query := "SELECT id, user_id, provider, external_id, COALESCE(label, ''), created_at " +
		"FROM user_oauth_connections WHERE provider = " + p(1) + " AND external_id = " + p(2)
	row := d.SQL().QueryRowContext(ctx, query, provider, externalID)
	var c OAuthConnection
	if err := row.Scan(&c.ID, &c.UserID, &c.Provider, &c.ExternalID, &c.Label, &c.CreatedAt); err != nil {
		return nil, translateError(err)
	}
	return &c, nil
}

// GetOAuthConnection returns a user's link for one provider.
func (d *DB) GetOAuthConnection(ctx context.Context, userID, provider string) (*OAuthConnection, error) {
	p := d.dialect.Placeholder
	query := "SELECT id, user_id, provider, external_id, COALESCE(label, ''), created_at " +
		"FROM user_oauth_connections WHERE user_id = " + p(1) + " AND provider = " + p(2)
	row := d.SQL().QueryRowContext(ctx, query, userID, provider)
	var c OAuthConnection
	if err := row.Scan(&c.ID, &c.UserID, &c.Provider, &c.ExternalID, &c.Label, &c.CreatedAt); err != nil {
		return nil, translateError(err)
	}
	return &c, nil
}

// UpsertOAuthConnection creates or replaces a provider link for a user.
func (d *DB) UpsertOAuthConnection(ctx context.Context, userID, provider, externalID, label string) (*OAuthConnection, error) {
	existing, err := d.GetOAuthConnection(ctx, userID, provider)
	if err == nil {
		p := d.dialect.Placeholder
		update := "UPDATE user_oauth_connections SET external_id = " + p(1) + ", label = " + p(2) +
			" WHERE id = " + p(3)
		if _, err := d.SQL().ExecContext(ctx, update, externalID, label, existing.ID); err != nil {
			return nil, translateError(err)
		}
		existing.ExternalID = externalID
		existing.Label = label
		return existing, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("oauth connection id: %w", err)
	}
	p := d.dialect.Placeholder
	insert := "INSERT INTO user_oauth_connections (id, user_id, provider, external_id, label) VALUES (" +
		p(1) + ", " + p(2) + ", " + p(3) + ", " + p(4) + ", " + p(5) + ")"
	if _, err := d.SQL().ExecContext(ctx, insert, id.String(), userID, provider, externalID, label); err != nil {
		return nil, translateError(err)
	}
	return d.GetOAuthConnection(ctx, userID, provider)
}

// DeleteOAuthConnection removes a provider link.
func (d *DB) DeleteOAuthConnection(ctx context.Context, userID, provider string) error {
	p := d.dialect.Placeholder
	query := "DELETE FROM user_oauth_connections WHERE user_id = " + p(1) + " AND provider = " + p(2)
	res, err := d.SQL().ExecContext(ctx, query, userID, provider)
	if err != nil {
		return translateError(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return translateError(err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// CountOAuthConnections returns how many providers a user has linked.
func (d *DB) CountOAuthConnections(ctx context.Context, userID string) (int, error) {
	p := d.dialect.Placeholder
	query := "SELECT COUNT(*) FROM user_oauth_connections WHERE user_id = " + p(1)
	var n int
	if err := d.SQL().QueryRowContext(ctx, query, userID).Scan(&n); err != nil {
		return 0, translateError(err)
	}
	return n, nil
}