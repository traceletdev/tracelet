package rules

import (
	"os"
	"path/filepath"
	"testing"

	"tracelet/internal/config"
	"tracelet/internal/stats"
)

func TestParseSize(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"35kb", 35 * 1024},
		{"10KB", 10 * 1024},
		{"1kib", 1024},
		{"500b", 500},
		{"50", 50}, // no unit → raw bytes
	}
	for _, c := range cases {
		if got := parseSize(c.in); got != c.want {
			t.Errorf("parseSize(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestHumanBytes(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{500, "500B"},
		{1024, "1KB"},
		{1536, "1.5KB"},
		{35 * 1024, "35KB"},
	}
	for _, c := range cases {
		if got := humanBytes(c.in); got != c.want {
			t.Errorf("humanBytes(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func routeCfg(budget string) config.Config {
	return config.Config{
		Rules:   map[string]string{"route-initial-js": "error"},
		Budgets: map[string]config.Budget{"default": {InitialJS: budget}},
	}
}

func TestRouteInitialJS(t *testing.T) {
	cfg := routeCfg("10kb") // 10240 bytes

	// Over budget → one error.
	over := RouteInitialJS(cfg, stats.Stats{Routes: []stats.RouteStat{{Path: "/", JSGzipBytes: 20000}}})
	if len(over) != 1 || over[0].Level != "error" || over[0].Route != "/" {
		t.Fatalf("over-budget: got %+v", over)
	}

	// Under budget → no results.
	under := RouteInitialJS(cfg, stats.Stats{Routes: []stats.RouteStat{{Path: "/", JSGzipBytes: 5000}}})
	if len(under) != 0 {
		t.Fatalf("under-budget: expected 0 results, got %+v", under)
	}

	// No stats → single info message.
	none := RouteInitialJS(cfg, stats.Stats{})
	if len(none) != 1 || none[0].Level != "info" {
		t.Fatalf("no-stats: expected one info result, got %+v", none)
	}
}

// writeTemp writes content to a temp file and returns its path.
func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestUnoptimizedImage(t *testing.T) {
	bad := writeTemp(t, "bad.html", `<img src="a.png">`)
	if got := UnoptimizedImage(bad, "warn"); len(got) != 1 {
		t.Errorf("unoptimized <img>: expected 1 result, got %+v", got)
	}

	good := writeTemp(t, "good.html", `<img src="a.png" width="10" height="10" loading="lazy">`)
	if got := UnoptimizedImage(good, "warn"); len(got) != 0 {
		t.Errorf("optimized <img>: expected 0 results, got %+v", got)
	}
}

func TestFontDisplay(t *testing.T) {
	missing := writeTemp(t, "a.css", `@font-face { font-family: X; src: url(x.woff2); }`)
	if got := FontDisplay(missing, "info"); len(got) != 1 {
		t.Errorf("missing font-display: expected 1 result, got %+v", got)
	}

	present := writeTemp(t, "b.css", `@font-face { font-family: X; font-display: swap; }`)
	if got := FontDisplay(present, "info"); len(got) != 0 {
		t.Errorf("font-display: swap present: expected 0 results, got %+v", got)
	}
}

func TestMissingAltText(t *testing.T) {
	missing := writeTemp(t, "a.html", `<img src="x.png">`)
	if got := MissingAltText(missing, "warn"); len(got) != 1 {
		t.Errorf("missing alt: expected 1 result, got %+v", got)
	}

	present := writeTemp(t, "b.html", `<img src="x.png" alt="a cat">`)
	if got := MissingAltText(present, "warn"); len(got) != 0 {
		t.Errorf("alt present: expected 0 results, got %+v", got)
	}
}
