package proxy

import "testing"

func TestDeploymentLogID_PrefersID(t *testing.T) {
	if got := deploymentLogID(Deployment{ID: "dep-1", Name: "fallback-name"}); got != "dep-1" {
		t.Errorf("deploymentLogID = %q, want dep-1", got)
	}
	if got := deploymentLogID(Deployment{Name: "synth"}); got != "synth" {
		t.Errorf("deploymentLogID = %q, want synth", got)
	}
}