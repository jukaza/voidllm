package provider_test

import (
	"testing"

	"github.com/voidmind-io/voidllm/internal/provider"
)

func TestOptionalCostFields(t *testing.T) {
	t.Parallel()

	in, out, cached, write := provider.OptionalCostFields(&provider.CostRef{
		In: 3, Out: 15, CachedIn: 0.3, CacheWrite: 3.75,
	})
	if in == nil || *in != 3 || out == nil || *out != 15 {
		t.Fatalf("in/out = %v, %v", in, out)
	}
	if cached == nil || *cached != 0.3 {
		t.Fatalf("cached = %v, want 0.3", cached)
	}
	if write == nil || *write != 3.75 {
		t.Fatalf("write = %v, want 3.75", write)
	}

	_, _, cached2, write2 := provider.OptionalCostFields(&provider.CostRef{In: 1, Out: 2})
	if cached2 != nil || write2 != nil {
		t.Fatalf("zero cache fields should be nil, got cached=%v write=%v", cached2, write2)
	}
}