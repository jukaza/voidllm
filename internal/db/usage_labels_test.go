package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/voidmind-io/voidllm/pkg/keygen"
)


func seedUserForLabels(t *testing.T, d *DB, email, displayName string) (id, label string) {
	t.Helper()
	user := mustCreateUser(t, d, CreateUserParams{Email: email, DisplayName: displayName})
	return user.ID, user.DisplayName
}

func seedCreatorUser(t *testing.T, d *DB, email string) string {
	t.Helper()
	user := mustCreateUser(t, d, CreateUserParams{Email: email, DisplayName: "creator"})
	return user.ID
}

func seedAPIKeyForLabels(t *testing.T, d *DB, keyName, createdBy string) (id, label string) {
	t.Helper()
	plaintext := "vl_uk_deadbeefdeadbeefdeadbeefdeadbeefdeadbeef00"
	params := CreateAPIKeyParams{
		KeyHash:   keygen.Hash(plaintext+keyName, testHMACSecret),
		KeyHint:   keygen.Hint(plaintext),
		KeyType:   keygen.KeyTypeUser,
		Name:      keyName,
		CreatedBy: createdBy,
	}
	key := mustCreateAPIKey(t, d, params)
	if keyName != "" {
		label = keyName
	} else {
		label = keygen.Hint(plaintext)
	}
	return key.ID, label
}

func TestResolveGroupLabels_NonResolvableDimensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		groupBy string
	}{
		{name: "empty groupBy", groupBy: ""},
		{name: "model", groupBy: "model"},
		{name: "day", groupBy: "day"},
		{name: "hour", groupBy: "hour"},
		{name: "server", groupBy: "server"},
		{name: "tool", groupBy: "tool"},
		{name: "status", groupBy: "status"},
		{name: "unknown value", groupBy: "unknown_xyz"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := openMigratedDB(t)

			got, err := d.ResolveGroupLabels(context.Background(), tc.groupBy, []string{"any-id"})
			if err != nil {
				t.Fatalf("ResolveGroupLabels(%q) error = %v, want nil", tc.groupBy, err)
			}
			if got == nil {
				t.Fatalf("ResolveGroupLabels(%q) returned nil map, want empty non-nil map", tc.groupBy)
			}
			if len(got) != 0 {
				t.Errorf("ResolveGroupLabels(%q) len = %d, want 0; got: %v", tc.groupBy, len(got), got)
			}
		})
	}
}

func TestResolveGroupLabels_EmptyIDsSlice(t *testing.T) {
	t.Parallel()

	d := openMigratedDB(t)

	for _, groupBy := range []string{"key", "user"} {
		got, err := d.ResolveGroupLabels(context.Background(), groupBy, []string{})
		if err != nil {
			t.Fatalf("groupBy=%q empty ids error = %v, want nil", groupBy, err)
		}
		if got == nil {
			t.Fatalf("groupBy=%q returned nil map, want empty non-nil map", groupBy)
		}
		if len(got) != 0 {
			t.Errorf("groupBy=%q len = %d, want 0", groupBy, len(got))
		}
	}
}

func TestResolveGroupLabels_OnlyEmptyStringIDs(t *testing.T) {
	t.Parallel()

	d := openMigratedDB(t)

	for _, groupBy := range []string{"key", "user"} {
		got, err := d.ResolveGroupLabels(context.Background(), groupBy, []string{"", "", ""})
		if err != nil {
			t.Fatalf("groupBy=%q all-empty ids error = %v, want nil", groupBy, err)
		}
		if got == nil {
			t.Fatalf("groupBy=%q returned nil map, want empty non-nil map", groupBy)
		}
		if len(got) != 0 {
			t.Errorf("groupBy=%q len = %d, want 0", groupBy, len(got))
		}
	}
}

func TestResolveGroupLabels_User(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T, d *DB) (ids []string, wantLabels map[string]string)
	}{
		{
			name: "single user resolves to display_name",
			setup: func(t *testing.T, d *DB) ([]string, map[string]string) {
				t.Helper()
				id, label := seedUserForLabels(t, d, "alice@example.com", "Alice Smith")
				return []string{id}, map[string]string{id: label}
			},
		},
		{
			name: "multiple users resolve correctly",
			setup: func(t *testing.T, d *DB) ([]string, map[string]string) {
				t.Helper()
				id1, label1 := seedUserForLabels(t, d, "bob@example.com", "Bob Jones")
				id2, label2 := seedUserForLabels(t, d, "carol@example.com", "Carol White")
				return []string{id1, id2}, map[string]string{id1: label1, id2: label2}
			},
		},
		{
			name: "non-existent user id is absent",
			setup: func(t *testing.T, d *DB) ([]string, map[string]string) {
				t.Helper()
				id, label := seedUserForLabels(t, d, "dave@example.com", "Dave Brown")
				ghost := "00000000-0000-0000-0000-000000000088"
				return []string{id, ghost}, map[string]string{id: label}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := openMigratedDB(t)
			ids, wantLabels := tc.setup(t, d)

			got, err := d.ResolveGroupLabels(context.Background(), "user", ids)
			if err != nil {
				t.Fatalf("ResolveGroupLabels(user) error = %v, want nil", err)
			}
			if len(got) != len(wantLabels) {
				t.Fatalf("len(got) = %d, want %d; got: %v", len(got), len(wantLabels), got)
			}
			for id, wantLabel := range wantLabels {
				if got[id] != wantLabel {
					t.Errorf("got[%q] = %q, want %q", id, got[id], wantLabel)
				}
			}
		})
	}
}

func TestResolveGroupLabels_Key_WithName(t *testing.T) {
	t.Parallel()

	d := openMigratedDB(t)
	creatorID := seedCreatorUser(t, d, "creator-key-named@example.com")

	plaintext := "vl_uk_aabbccddaabbccddaabbccddaabbccddaabbccddaabb"
	params := CreateAPIKeyParams{
		KeyHash:   keygen.Hash(plaintext, testHMACSecret),
		KeyHint:   keygen.Hint(plaintext),
		KeyType:   keygen.KeyTypeUser,
		Name:      "My Named Key",
		CreatedBy: creatorID,
	}
	key := mustCreateAPIKey(t, d, params)

	got, err := d.ResolveGroupLabels(context.Background(), "key", []string{key.ID})
	if err != nil {
		t.Fatalf("ResolveGroupLabels(key) error = %v, want nil", err)
	}
	if got[key.ID] != "My Named Key" {
		t.Errorf("label = %q, want %q (key name)", got[key.ID], "My Named Key")
	}
}

func TestResolveGroupLabels_Key_WithoutName_FallsBackToHint(t *testing.T) {
	t.Parallel()

	d := openMigratedDB(t)
	creatorID := seedCreatorUser(t, d, "creator-key-unnamed@example.com")

	plaintext := "vl_uk_11223344112233441122334411223344112233441122"
	hint := keygen.Hint(plaintext)
	params := CreateAPIKeyParams{
		KeyHash:   keygen.Hash(plaintext, testHMACSecret),
		KeyHint:   hint,
		KeyType:   keygen.KeyTypeUser,
		Name:      "",
		CreatedBy: creatorID,
	}
	key := mustCreateAPIKey(t, d, params)

	got, err := d.ResolveGroupLabels(context.Background(), "key", []string{key.ID})
	if err != nil {
		t.Fatalf("ResolveGroupLabels(key) error = %v, want nil", err)
	}
	if got[key.ID] != hint {
		t.Errorf("label = %q, want hint %q (name was empty)", got[key.ID], hint)
	}
}

func TestResolveGroupLabels_Key_SoftDeleted_StillResolves(t *testing.T) {
	t.Parallel()

	d := openMigratedDB(t)
	creatorID := seedCreatorUser(t, d, "creator-key-softdel@example.com")

	plaintext := "vl_uk_ffeeddccffeeddccffeeddccffeeddccffeeddccffee"
	params := CreateAPIKeyParams{
		KeyHash:   keygen.Hash(plaintext, testHMACSecret),
		KeyHint:   keygen.Hint(plaintext),
		KeyType:   keygen.KeyTypeUser,
		Name:      "Ghost Key",
		CreatedBy: creatorID,
	}
	key := mustCreateAPIKey(t, d, params)

	if err := d.DeleteAPIKey(context.Background(), key.ID); err != nil {
		t.Fatalf("DeleteAPIKey: %v", err)
	}

	got, err := d.ResolveGroupLabels(context.Background(), "key", []string{key.ID})
	if err != nil {
		t.Fatalf("ResolveGroupLabels(key) soft-deleted error = %v, want nil", err)
	}
	if got[key.ID] != "Ghost Key" {
		t.Errorf("label = %q, want %q (soft-deleted key must still resolve)", got[key.ID], "Ghost Key")
	}
}

func TestResolveGroupLabels_Key_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		keyName     string
		wantLabelFn func(hint string) string
	}{
		{
			name:        "named key returns key name",
			keyName:     "Production Key",
			wantLabelFn: func(_ string) string { return "Production Key" },
		},
		{
			name:        "unnamed key returns key hint",
			keyName:     "",
			wantLabelFn: func(hint string) string { return hint },
		},
	}

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d := openMigratedDB(t)
			creatorID := seedCreatorUser(t, d, fmt.Sprintf("creator-key-td-%d@example.com", i))

			plaintext := fmt.Sprintf("vl_uk_%048d", i)
			hint := keygen.Hint(plaintext)

			params := CreateAPIKeyParams{
				KeyHash:   keygen.Hash(plaintext, testHMACSecret),
				KeyHint:   hint,
				KeyType:   keygen.KeyTypeUser,
				Name:      tc.keyName,
				CreatedBy: creatorID,
			}
			key := mustCreateAPIKey(t, d, params)

			got, err := d.ResolveGroupLabels(context.Background(), "key", []string{key.ID})
			if err != nil {
				t.Fatalf("ResolveGroupLabels(key) error = %v, want nil", err)
			}
			wantLabel := tc.wantLabelFn(hint)
			if got[key.ID] != wantLabel {
				t.Errorf("label = %q, want %q", got[key.ID], wantLabel)
			}
		})
	}
}

func TestResolveGroupLabels_Deduplication(t *testing.T) {
	t.Parallel()

	d := openMigratedDB(t)
	id, _ := seedUserForLabels(t, d, "dedup-user@example.com", "Dedup User")

	got, err := d.ResolveGroupLabels(context.Background(), "user", []string{id, id, id})
	if err != nil {
		t.Fatalf("ResolveGroupLabels error = %v, want nil", err)
	}
	if len(got) != 1 {
		t.Errorf("len(got) = %d, want 1 (same id passed three times)", len(got))
	}
}

func TestResolveGroupLabels_EmptyStringIDsFiltered(t *testing.T) {
	t.Parallel()

	d := openMigratedDB(t)
	id, label := seedUserForLabels(t, d, "mixed-user@example.com", "Mixed User")

	got, err := d.ResolveGroupLabels(context.Background(), "user", []string{"", id, ""})
	if err != nil {
		t.Fatalf("ResolveGroupLabels error = %v, want nil", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[id] != label {
		t.Errorf("got[%q] = %q, want %q", id, got[id], label)
	}
}

func TestResolveGroupLabels_AllResolvableDimensions(t *testing.T) {
	t.Parallel()

	d := openMigratedDB(t)
	ctx := context.Background()

	userID, userLabel := seedUserForLabels(t, d, "dimension@example.com", "Dimension User")
	creatorID := seedCreatorUser(t, d, "dimension-creator@example.com")
	plaintext := "vl_uk_99887766998877669988776699887766998877669988"
	keyParams := CreateAPIKeyParams{
		KeyHash:   keygen.Hash(plaintext, testHMACSecret),
		KeyHint:   keygen.Hint(plaintext),
		KeyType:   keygen.KeyTypeUser,
		Name:      "Dimension Key",
		CreatedBy: creatorID,
	}
	key := mustCreateAPIKey(t, d, keyParams)
	keyLabel := "Dimension Key"

	tests := []struct {
		groupBy   string
		id        string
		wantLabel string
	}{
		{groupBy: "user", id: userID, wantLabel: userLabel},
		{groupBy: "key", id: key.ID, wantLabel: keyLabel},
	}

	for _, tc := range tests {
		t.Run("groupBy="+tc.groupBy, func(t *testing.T) {
			t.Parallel()

			got, err := d.ResolveGroupLabels(ctx, tc.groupBy, []string{tc.id})
			if err != nil {
				t.Fatalf("ResolveGroupLabels(%q) error = %v, want nil", tc.groupBy, err)
			}
			if got[tc.id] != tc.wantLabel {
				t.Errorf("label = %q, want %q", got[tc.id], tc.wantLabel)
			}
		})
	}
}


