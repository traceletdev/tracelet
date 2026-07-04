package reporters

import (
	"fmt"
	"io"

	"tracelet/internal/ci"
	"tracelet/internal/lint"
	"tracelet/internal/stats"
)

func PrintMarkdown(w io.Writer, results lint.Results, st stats.Stats, base *ci.Baseline, diff *ci.Diff) {
	fmt.Fprintln(w, "| Route | JS(gzip) | Verdict |")
	fmt.Fprintln(w, "|---|---:|---|")
	for _, r := range st.Routes {
		verdict := "✅"
		for _, res := range results {
			if res.Route == r.Path && res.Level == "error" && res.RuleID == "route-initial-js" {
				verdict = "❌ over budget"
				break
			}
		}
		fmt.Fprintf(w, "| %s | %d | %s |\n", r.Path, r.JSGzipBytes, verdict)
	}
	if diff != nil {
		fmt.Fprintln(w, "\n**Diagnostics delta**")
		fmt.Fprintf(w, "- Errors: %d → %d\n", diff.ErrorsBefore, diff.ErrorsAfter)
		fmt.Fprintf(w, "- Warnings: %d → %d\n", diff.WarnsBefore, diff.WarnsAfter)
	}
}
