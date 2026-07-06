package backup

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/voidmind-io/voidllm/internal/db"
)

// ConfigPayload is the JSON format for config-scope cloud backups.
type ConfigPayload struct {
	Version    int                       `json:"version"`
	ExportedAt string                    `json:"exported_at"`
	Tables     map[string][]map[string]any `json:"tables"`
}

// configTableOrder lists operational tables in FK-safe delete/insert order.
var configTableOrder = []string{
	"model_route_steps",
	"model_deployments",
	"model_aliases",
	"provider_upstream_models",
	"provider_connections",
	"providers",
	"models",
	"output_schemas",
	"mcp_tool_blocklist",
	"mcp_server_tools",
	"mcp_servers",
	"settings",
}

var excludedSettingPrefixes = []string{
	"backup.s3",
	"backup.schedule",
	"backup.records",
}

func isExcludedSettingKey(key string) bool {
	for _, p := range excludedSettingPrefixes {
		if key == p || strings.HasPrefix(key, p+".") {
			return true
		}
	}
	return false
}

func dumpConfigTables(ctx context.Context, sqlDB *sql.DB, dialect string) (*ConfigPayload, error) {
	tables := make(map[string][]map[string]any, len(configTableOrder))
	for _, table := range configTableOrder {
		rows, err := queryTableRows(ctx, sqlDB, table)
		if err != nil {
			// Skip tables that do not exist on older schemas.
			if strings.Contains(err.Error(), "no such table") ||
				strings.Contains(err.Error(), "does not exist") {
				continue
			}
			return nil, err
		}
		if table == "settings" {
			rows = filterSettingsRows(rows)
		}
		tables[table] = rows
	}
	return &ConfigPayload{
		Version:    1,
		ExportedAt: nowRFC3339(),
		Tables:     tables,
	}, nil
}

func queryTableRows(ctx context.Context, sqlDB *sql.DB, table string) ([]map[string]any, error) {
	rows, err := sqlDB.QueryContext(ctx, "SELECT * FROM "+table)
	if err != nil {
		return nil, fmt.Errorf("query %s: %w", table, err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("columns %s: %w", table, err)
	}

	var out []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("scan %s: %w", table, err)
		}
		row := make(map[string]any, len(cols))
		for i, col := range cols {
			row[col] = normalizeCell(vals[i])
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate %s: %w", table, err)
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out, nil
}

func normalizeCell(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case []byte:
		return string(x)
	default:
		return x
	}
}

func filterSettingsRows(rows []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		key, _ := row["key"].(string)
		if isExcludedSettingKey(key) {
			continue
		}
		out = append(out, row)
	}
	return out
}

func restoreConfigTables(ctx context.Context, sqlDB *sql.DB, dialect db.Dialect, payload *ConfigPayload) error {
	if payload == nil || payload.Version != 1 {
		return fmt.Errorf("unsupported config backup version")
	}
	tx, err := sqlDB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// Delete in reverse FK order.
	for i := len(configTableOrder) - 1; i >= 0; i-- {
		table := configTableOrder[i]
		rows := payload.Tables[table]
		if rows == nil {
			continue
		}
		if table == "settings" {
			if err := deleteConfigSettings(ctx, tx, dialect); err != nil {
				return err
			}
		} else {
			if _, err := tx.ExecContext(ctx, "DELETE FROM "+table); err != nil {
				if strings.Contains(err.Error(), "no such table") ||
					strings.Contains(err.Error(), "does not exist") {
					continue
				}
				return fmt.Errorf("delete %s: %w", table, err)
			}
		}
	}

	// Insert in FK order.
	for _, table := range configTableOrder {
		rows := payload.Tables[table]
		if len(rows) == 0 {
			continue
		}
		if err := insertRows(ctx, tx, dialect, table, rows); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func deleteConfigSettings(ctx context.Context, tx *sql.Tx, dialect db.Dialect) error {
	rows, err := tx.QueryContext(ctx, "SELECT key FROM settings")
	if err != nil {
		return fmt.Errorf("list settings keys: %w", err)
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return err
		}
		if !isExcludedSettingKey(key) {
			keys = append(keys, key)
		}
	}
	p := dialect.Placeholder
	for _, key := range keys {
		q := "DELETE FROM settings WHERE key = " + p(1)
		if _, err := tx.ExecContext(ctx, q, key); err != nil {
			return fmt.Errorf("delete setting %s: %w", key, err)
		}
	}
	return rows.Err()
}

func insertRows(ctx context.Context, tx *sql.Tx, dialect db.Dialect, table string, rows []map[string]any) error {
	for _, row := range rows {
		cols := make([]string, 0, len(row))
		for k := range row {
			cols = append(cols, k)
		}
		sort.Strings(cols)
		vals := make([]any, 0, len(cols))
		for _, col := range cols {
			vals = append(vals, row[col])
		}
		placeholders := make([]string, len(cols))
		for i := range cols {
			placeholders[i] = dialect.Placeholder(i + 1)
		}
		q := fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES (%s)",
			table,
			strings.Join(cols, ","),
			strings.Join(placeholders, ","),
		)
		if _, err := tx.ExecContext(ctx, q, vals...); err != nil {
			if strings.Contains(err.Error(), "no such table") ||
				strings.Contains(err.Error(), "does not exist") {
				return nil
			}
			return fmt.Errorf("insert %s: %w", table, err)
		}
	}
	return nil
}

func encodeConfigPayload(p *ConfigPayload) ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

func decodeConfigPayload(data []byte) (*ConfigPayload, error) {
	var p ConfigPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}