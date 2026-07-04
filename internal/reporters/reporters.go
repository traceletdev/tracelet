package reporters

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"tracelet/internal/lint"
	"tracelet/internal/stats"
)

func PrintJSON(w io.Writer, results lint.Results, st stats.Stats) {
	payload := struct {
		Results lint.Results `json:"results"`
		Stats   stats.Stats  `json:"stats"`
	}{Results: results, Stats: st}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(payload)
}

func PrintTable(w io.Writer, results lint.Results, st stats.Stats) {
	// Header
	fmt.Fprintln(w, "Route        JS(gzip)  Verdict")
	for _, r := range st.Routes {
		verdict := "✅"
		// if any error on this route
		for _, res := range results {
			if res.Route == r.Path && strings.EqualFold(res.Level, "error") && res.RuleID == "route-initial-js" {
				verdict = "❌ over budget"
				break
			}
		}
		fmt.Fprintf(w, "%-12s %-8d %s\n", r.Path, r.JSGzipBytes, verdict)
	}
	if len(results) > 0 {
		fmt.Fprintln(w, "\nDiagnostics:")
		for _, res := range results {
			route := res.Route
			if route == "" {
				route = "-"
			}
			icon := map[string]string{"info": "ℹ️ ", "warn": "⚠️ ", "error": "❌ "}[strings.ToLower(res.Level)]
			fmt.Fprintf(w, "%s[%s] %s — %s\n", icon, res.RuleID, route, res.Detail)
		}
	}
}
