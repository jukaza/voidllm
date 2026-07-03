package db

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/voidmind-io/voidllm/internal/config"
)

func openMigratedDB(t *testing.T) *DB {
	t.Helper()

	// Replace characters that SQLite rejects in URI filenames.
	safeName := strings.NewReplacer("/", "_", " ", "_", "#", "_").Replace(t.Name())

	cfg := config.DatabaseConfig{
		Driver:          "sqlite",
		DSN:             "file:" + safeName + "?mode=memory&cache=private",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
	}
	ctx := context.Background()
	d, err := Open(ctx, cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	if err := RunMigrations(ctx, d.SQL(), d.Dialect(), slog.Default()); err != nil {
		t.Fatalf("RunMigrations() error = %v", err)
	}

	return d
}

func ptr[T any](v T) *T {
	return &v
}

func mustCreateOrg(t *testing.T, d *DB, params CreateOrgParams) *Org {
	t.Helper()
	org, err := d.CreateOrg(context.Background(), params)
	if err != nil {
		t.Fatalf("mustCreateOrg(%q): %v", params.Slug, err)
	}
	return org
}

func mustCreateTeam(t *testing.T, d *DB, params CreateTeamParams) *Team {
	t.Helper()
	team, err := d.CreateTeam(context.Background(), params)
	if err != nil {
		t.Fatalf("mustCreateTeam(%q): %v", params.Slug, err)
	}
	return team
}

func mustCreateMembership(t *testing.T, d *DB, params CreateOrgMembershipParams) *OrgMembership {
	t.Helper()
	m, err := d.CreateOrgMembership(context.Background(), params)
	if err != nil {
		t.Fatalf("mustCreateMembership: %v", err)
	}
	return m
}

func mustCreateServiceAccount(t *testing.T, d *DB, params CreateServiceAccountParams) *ServiceAccount {
	t.Helper()
	sa, err := d.CreateServiceAccount(context.Background(), params)
	if err != nil {
		t.Fatalf("mustCreateServiceAccount: %v", err)
	}
	return sa
}


