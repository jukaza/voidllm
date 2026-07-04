package db

import (
	"context"
	"errors"
	"testing"
	"time"
)

func mustCreateProvider(t *testing.T, d *DB, name string) *Provider {
	t.Helper()
	p, err := d.CreateProvider(context.Background(), CreateProviderParams{Name: name, Status: "active"})
	if err != nil {
		t.Fatalf("mustCreateProvider(%q): %v", name, err)
	}
	return p
}

func TestProviderConnection_CRUDAndReorder(t *testing.T) {
	d := openMigratedDB(t)
	ctx := context.Background()
	prov := mustCreateProvider(t, d, "conn-test-provider")

	c1, err := d.CreateProviderConnection(ctx, CreateProviderConnectionParams{
		ProviderID: prov.ID, Name: "key-a", Priority: 1,
	})
	if err != nil {
		t.Fatalf("CreateProviderConnection c1: %v", err)
	}
	c2, err := d.CreateProviderConnection(ctx, CreateProviderConnectionParams{
		ProviderID: prov.ID, Name: "key-b",
	})
	if err != nil {
		t.Fatalf("CreateProviderConnection c2: %v", err)
	}
	if c2.Priority != 2 {
		t.Errorf("auto priority = %d, want 2", c2.Priority)
	}

	list, err := d.ListProviderConnections(ctx, prov.ID, false)
	if err != nil {
		t.Fatalf("ListProviderConnections: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}

	inactive := false
	renamed := "key-b-renamed"
	updated, err := d.UpdateProviderConnection(ctx, c2.ID, UpdateProviderConnectionParams{
		Name:     &renamed,
		IsActive: &inactive,
	})
	if err != nil {
		t.Fatalf("UpdateProviderConnection: %v", err)
	}
	if updated.Name != renamed || updated.IsActive {
		t.Errorf("update: name=%q active=%v", updated.Name, updated.IsActive)
	}

	activeOnly, err := d.ListProviderConnections(ctx, prov.ID, true)
	if err != nil {
		t.Fatalf("ListProviderConnections activeOnly: %v", err)
	}
	if len(activeOnly) != 1 || activeOnly[0].ID != c1.ID {
		t.Errorf("activeOnly = %+v, want only c1", activeOnly)
	}

	if err := d.ReorderProviderConnections(ctx, prov.ID, []string{c2.ID, c1.ID}); err != nil {
		t.Fatalf("ReorderProviderConnections: %v", err)
	}
	reordered, err := d.ListProviderConnections(ctx, prov.ID, false)
	if err != nil {
		t.Fatalf("List after reorder: %v", err)
	}
	if len(reordered) != 2 || reordered[0].ID != c2.ID || reordered[0].Priority != 1 {
		t.Errorf("reordered[0] = %+v, want c2 priority 1", reordered[0])
	}

	until := time.Now().UTC().Add(2 * time.Minute)
	locked, err := d.SetProviderConnectionModelLock(ctx, c1.ID, "gpt-4o", until)
	if err != nil {
		t.Fatalf("SetProviderConnectionModelLock: %v", err)
	}
	if locked.ModelLocks["gpt-4o"] == "" {
		t.Errorf("model lock missing for gpt-4o: %+v", locked.ModelLocks)
	}

	cleared, err := d.ClearProviderConnectionLocks(ctx, c1.ID)
	if err != nil {
		t.Fatalf("ClearProviderConnectionLocks: %v", err)
	}
	if len(cleared.ModelLocks) != 0 || cleared.TestStatus != "active" {
		t.Errorf("cleared = status %q locks %v", cleared.TestStatus, cleared.ModelLocks)
	}

	if err := d.DeleteProviderConnection(ctx, c2.ID); err != nil {
		t.Fatalf("DeleteProviderConnection: %v", err)
	}
	if _, err := d.GetProviderConnection(ctx, c2.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get deleted connection: err = %v, want ErrNotFound", err)
	}
}

func TestProviderUpstreamModel_UpsertAndUnique(t *testing.T) {
	d := openMigratedDB(t)
	ctx := context.Background()
	prov := mustCreateProvider(t, d, "upstream-test-provider")

	in := 1.5
	out := 2.0
	m1, err := d.UpsertProviderUpstreamModel(ctx, UpsertProviderUpstreamModelParams{
		ProviderID: prov.ID, UpstreamID: "gpt-4o", CostInputPer1M: &in, CostOutputPer1M: &out,
	})
	if err != nil {
		t.Fatalf("Upsert first: %v", err)
	}
	if m1.DisplayName != "gpt-4o" || !m1.IsEnabled {
		t.Errorf("m1 = %+v", m1)
	}

	in2 := 3.0
	m2, err := d.UpsertProviderUpstreamModel(ctx, UpsertProviderUpstreamModelParams{
		ProviderID: prov.ID, UpstreamID: "gpt-4o", CostInputPer1M: &in2,
	})
	if err != nil {
		t.Fatalf("Upsert second: %v", err)
	}
	if m2.ID != m1.ID {
		t.Errorf("upsert created duplicate id %s vs %s", m2.ID, m1.ID)
	}
	if m2.CostInputPer1M == nil || *m2.CostInputPer1M != 3.0 {
		t.Errorf("cost not updated: %+v", m2.CostInputPer1M)
	}

	list, err := d.ListProviderUpstreamModels(ctx, prov.ID, false)
	if err != nil {
		t.Fatalf("ListProviderUpstreamModels: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(list))
	}

	if err := d.DeleteProviderUpstreamModel(ctx, m1.ID); err != nil {
		t.Fatalf("DeleteProviderUpstreamModel: %v", err)
	}
}

func TestModelRouteSteps_Replace(t *testing.T) {
	d := openMigratedDB(t)
	ctx := context.Background()
	prov := mustCreateProvider(t, d, "route-test-provider")
	model := mustCreateModel(t, d, "combo-product")

	steps, err := d.ReplaceModelRouteSteps(ctx, model.ID, []ModelRouteStepInput{
		{ProviderID: prov.ID, UpstreamModel: "glm-4.7", IsEnabled: true},
		{ProviderID: prov.ID, UpstreamModel: "gpt-4o", IsEnabled: false},
	})
	if err != nil {
		t.Fatalf("ReplaceModelRouteSteps: %v", err)
	}
	if len(steps) != 2 {
		t.Fatalf("len(steps) = %d, want 2", len(steps))
	}
	if steps[0].Position != 0 || steps[0].UpstreamModel != "glm-4.7" {
		t.Errorf("step[0] = %+v", steps[0])
	}

	enabled, err := d.ListModelRouteSteps(ctx, model.ID, true)
	if err != nil {
		t.Fatalf("ListModelRouteSteps enabledOnly: %v", err)
	}
	if len(enabled) != 1 || enabled[0].UpstreamModel != "glm-4.7" {
		t.Errorf("enabled steps = %+v", enabled)
	}

	replaced, err := d.ReplaceModelRouteSteps(ctx, model.ID, []ModelRouteStepInput{
		{ProviderID: prov.ID, UpstreamModel: "mini-max", IsEnabled: true},
	})
	if err != nil {
		t.Fatalf("Replace again: %v", err)
	}
	if len(replaced) != 1 || replaced[0].UpstreamModel != "mini-max" {
		t.Errorf("replaced = %+v", replaced)
	}
}

func TestDeleteProvider_CascadesDependentData(t *testing.T) {
	d := openMigratedDB(t)
	ctx := context.Background()
	prov := mustCreateProvider(t, d, "cascade-delete-provider")
	model := mustCreateModel(t, d, "cascade-product")

	if _, err := d.CreateProviderConnection(ctx, CreateProviderConnectionParams{
		ProviderID: prov.ID, Name: "key-a", Priority: 1,
	}); err != nil {
		t.Fatalf("CreateProviderConnection: %v", err)
	}
	if _, err := d.UpsertProviderUpstreamModel(ctx, UpsertProviderUpstreamModelParams{
		ProviderID: prov.ID, UpstreamID: "glm-4.7",
	}); err != nil {
		t.Fatalf("UpsertProviderUpstreamModel: %v", err)
	}
	if _, err := d.ReplaceModelRouteSteps(ctx, model.ID, []ModelRouteStepInput{
		{ProviderID: prov.ID, UpstreamModel: "glm-4.7", IsEnabled: true},
	}); err != nil {
		t.Fatalf("ReplaceModelRouteSteps: %v", err)
	}

	if err := d.DeleteProvider(ctx, prov.ID); err != nil {
		t.Fatalf("DeleteProvider: %v", err)
	}
	if _, err := d.GetProvider(ctx, prov.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetProvider after delete: err = %v, want ErrNotFound", err)
	}

	upstream, err := d.ListProviderUpstreamModels(ctx, prov.ID, false)
	if err != nil {
		t.Fatalf("ListProviderUpstreamModels: %v", err)
	}
	if len(upstream) != 0 {
		t.Fatalf("upstream inventory len = %d, want 0", len(upstream))
	}

	conns, err := d.ListProviderConnections(ctx, prov.ID, false)
	if err != nil {
		t.Fatalf("ListProviderConnections: %v", err)
	}
	if len(conns) != 0 {
		t.Fatalf("connections len = %d, want 0", len(conns))
	}

	routes, err := d.ListModelRouteSteps(ctx, model.ID, false)
	if err != nil {
		t.Fatalf("ListModelRouteSteps: %v", err)
	}
	if len(routes) != 0 {
		t.Fatalf("route steps len = %d, want 0", len(routes))
	}

	allUpstream, err := d.ListAllProviderUpstreamModels(ctx, false)
	if err != nil {
		t.Fatalf("ListAllProviderUpstreamModels: %v", err)
	}
	for _, m := range allUpstream {
		if m.ProviderID == prov.ID {
			t.Fatalf("ListAllProviderUpstreamModels still contains provider %s", prov.ID)
		}
	}
}