package backup

import (
	"compress/gzip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/voidmind-io/voidllm/internal/config"
	"github.com/voidmind-io/voidllm/internal/db"
)

// FullDumper produces a gzip-compressed full database backup stream.
type FullDumper struct {
	Driver string
	DSN    string
}

func NewFullDumper(cfg config.DatabaseConfig) *FullDumper {
	return &FullDumper{Driver: cfg.Driver, DSN: cfg.DSN}
}

func (d *FullDumper) Dump(ctx context.Context) (io.ReadCloser, error) {
	pr, pw := io.Pipe()
	go func() {
		var err error
		defer func() {
			if err != nil {
				_ = pw.CloseWithError(err)
			} else {
				_ = pw.Close()
			}
		}()
		gz := gzip.NewWriter(pw)
		defer gz.Close()

		switch d.Driver {
		case "sqlite":
			err = d.dumpSQLite(ctx, gz)
		case "postgres":
			err = d.dumpPostgres(ctx, gz)
		default:
			err = fmt.Errorf("unsupported driver %q", d.Driver)
		}
	}()
	return pr, nil
}

func (d *FullDumper) dumpSQLite(ctx context.Context, w io.Writer) error {
	tmpDir, err := os.MkdirTemp("", "voidllm-backup-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	dst := filepath.Join(tmpDir, "backup.db")
	if _, err := exec.LookPath("sqlite3"); err == nil {
		out, cmdErr := exec.CommandContext(ctx, "sqlite3", d.DSN, fmt.Sprintf(".backup '%s'", dst)).CombinedOutput()
		if cmdErr == nil {
			return copyFileToWriter(dst, w)
		}
		_ = out
	}

	db, err := sql.Open("sqlite", d.DSN)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", dst); err != nil {
		return fmt.Errorf("vacuum into: %w", err)
	}
	return copyFileToWriter(dst, w)
}

func copyFileToWriter(path string, w io.Writer) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err
}

func (d *FullDumper) dumpPostgres(ctx context.Context, w io.Writer) error {
	if _, err := exec.LookPath("pg_dump"); err != nil {
		return fmt.Errorf("pg_dump not found in PATH")
	}
	cmd := exec.CommandContext(ctx, "pg_dump",
		"--no-owner", "--no-acl", "--clean", "--if-exists",
		d.DSN,
	)
	cmd.Stdout = w
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pg_dump: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (d *FullDumper) Restore(ctx context.Context, r io.Reader) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()

	switch d.Driver {
	case "sqlite":
		return d.restoreSQLite(ctx, gz)
	case "postgres":
		return d.restorePostgres(ctx, gz)
	default:
		return fmt.Errorf("unsupported driver %q", d.Driver)
	}
}

func (d *FullDumper) restoreSQLite(ctx context.Context, r io.Reader) error {
	tmpDir, err := os.MkdirTemp("", "voidllm-restore-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	backupPath := filepath.Join(tmpDir, "restore.db")
	f, err := os.Create(backupPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return err
	}
	f.Close()

	src, err := sql.Open("sqlite", backupPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := sql.Open("sqlite", d.DSN)
	if err != nil {
		return err
	}
	defer dst.Close()

	tables, err := listTables(ctx, src)
	if err != nil {
		return err
	}
	tx, err := dst.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	for i := len(tables) - 1; i >= 0; i-- {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+tables[i]); err != nil {
			return err
		}
	}
	for _, table := range tables {
		rows, err := queryTableRows(ctx, src, table)
		if err != nil {
			return err
		}
		if err := insertRows(ctx, tx, db.SQLiteDialect{}, table, rows); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (d *FullDumper) restorePostgres(ctx context.Context, r io.Reader) error {
	if _, err := exec.LookPath("psql"); err != nil {
		return fmt.Errorf("psql not found in PATH")
	}
	cmd := exec.CommandContext(ctx, "psql", "--single-transaction", d.DSN)
	cmd.Stdin = r
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("psql restore: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func listTables(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx,
		"SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func backupFileName(backupType, driver string) string {
	ts := time.Now().UTC().Format("20060102_150405")
	ext := "json.gz"
	if backupType == TypeFull {
		if driver == "postgres" {
			ext = "sql.gz"
		} else {
			ext = "db.gz"
		}
	}
	return fmt.Sprintf("voidllm_%s_%s_%s.%s", backupType, driver, ts, ext)
}

func s3Key(prefix, fileName string) string {
	prefix = strings.Trim(prefix, "/")
	now := time.Now().UTC()
	datePath := fmt.Sprintf("%04d/%02d/%02d", now.Year(), now.Month(), now.Day())
	if prefix == "" {
		return datePath + "/" + fileName
	}
	return prefix + "/" + datePath + "/" + fileName
}