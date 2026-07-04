package ci

import (
	"testing"

	"tracelet/internal/lint"
	"tracelet/internal/stats"
)

func TestComputeDiff(t *testing.T) {
	prev := lint.Results{{Level: "error"}, {Level: "warn"}, {Level: "warn"}}
	curr := lint.Results{{Level: "error"}, {Level: "error"}, {Level: "warn"}}

	d := ComputeDiff(prev, curr)
	if d.ErrorsBefore != 1 || d.ErrorsAfter != 2 {
		t.Errorf("errors: before=%d after=%d, want 1/2", d.ErrorsBefore, d.ErrorsAfter)
	}
	if d.WarnsBefore != 2 || d.WarnsAfter != 1 {
		t.Errorf("warns: before=%d after=%d, want 2/1", d.WarnsBefore, d.WarnsAfter)
	}
}

func TestBaselineRoundTrip(t *testing.T) {
	// SaveBaseline does a relative MkdirAll(".tracelet"); isolate it in a temp dir.
	t.Chdir(t.TempDir())
	p := "baseline.json"
	in := Baseline{
		Results: lint.Results{{RuleID: "route-initial-js", Level: "error", Route: "/"}},
		Stats:   stats.Stats{Routes: []stats.RouteStat{{Path: "/", JSGzipBytes: 1234}}},
	}
	if err := SaveBaseline(p, in); err != nil {
		t.Fatalf("SaveBaseline: %v", err)
	}
	out, err := LoadBaseline(p)
	if err != nil {
		t.Fatalf("LoadBaseline: %v", err)
	}
	if len(out.Results) != 1 || out.Results[0].RuleID != "route-initial-js" {
		t.Errorf("results round-trip mismatch: %+v", out.Results)
	}
	if len(out.Stats.Routes) != 1 || out.Stats.Routes[0].JSGzipBytes != 1234 {
		t.Errorf("stats round-trip mismatch: %+v", out.Stats)
	}
}
