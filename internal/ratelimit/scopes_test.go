package ratelimit

import "testing"

func TestScopeKeys_UniquePrefixes(t *testing.T) {
	depID := "pm:foo"
	prov := ScopeProvider("abc")
	model := ScopeProductModel("foo")
	if prov == model {
		t.Fatalf("provider and product scopes must differ: %q", prov)
	}
	if prov == depID && depID == "prov:abc" {
		// dep collision only if id literally matches
	}
	if ScopeProvider("x") != "prov:x" {
		t.Fatal("unexpected provider scope prefix")
	}
	if ScopeProductModel("x") != "pm:x" {
		t.Fatal("unexpected product scope prefix")
	}
}

func TestUpstreamLimiter_ProductAndProviderScopes(t *testing.T) {
	u := NewUpstreamLimiter()
	modelScope := ScopeProductModel("gpt-test")
	provScope := ScopeProvider("prov-1")

	if !u.Allow(modelScope, 2, 0, 0) {
		t.Fatal("first allow model")
	}
	u.RecordRequest(modelScope)
	u.RecordRequest(modelScope)
	if u.Allow(modelScope, 2, 0, 0) {
		t.Fatal("model rpm should be exhausted")
	}
	if !u.Allow(provScope, 5, 0, 0) {
		t.Fatal("provider scope independent of model scope")
	}
}