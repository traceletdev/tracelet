package rules

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"tracelet/internal/config"
	"tracelet/internal/stats"
)

type Result = struct {
	RuleID string `json:"ruleId"`
	Level  string `json:"level"`
	Route  string `json:"route,omitempty"`
	File   string `json:"file,omitempty"`
	Detail string `json:"detail"`
}

// RouteInitialJS enforces per-route initial JS budgets using gzip byte counts.
func RouteInitialJS(cfg config.Config, st stats.Stats) []Result {
	lvl := cfg.Rules["route-initial-js"]
	var out []Result
	defaultBudget := parseSize(cfg.Budgets["default"].InitialJS)
	defaultThirdPartyBudget := parseSize(cfg.Budgets["default"].ThirdPartyJS)

	// If no stats available, provide helpful message
	if len(st.Routes) == 0 {
		return []Result{{
			RuleID: "route-initial-js",
			Level:  "info",
			Detail: fmt.Sprintf("No stats found. Run adapter: node adapters/%s-collect.js", cfg.Collect.Framework),
		}}
	}

	for _, r := range st.Routes {
		// Check initial JS budget
		budget := defaultBudget
		if b, ok := cfg.Budgets[r.Path]; ok && b.InitialJS != "" {
			budget = parseSize(b.InitialJS)
		}
		if r.JSGzipBytes > budget && budget > 0 {
			diff := r.JSGzipBytes - budget
			out = append(out, Result{
				RuleID: "route-initial-js",
				Level:  lvl,
				Route:  r.Path,
				Detail: humanBytes(r.JSGzipBytes) + " JS (over by " + humanBytes(diff) + ")",
			})
		}
		// Check third-party JS budget if configured
		thirdPartyBudget := defaultThirdPartyBudget
		if b, ok := cfg.Budgets[r.Path]; ok && b.ThirdPartyJS != "" {
			thirdPartyBudget = parseSize(b.ThirdPartyJS)
		}
		if r.ThirdPartyJS > thirdPartyBudget && thirdPartyBudget > 0 {
			diff := r.ThirdPartyJS - thirdPartyBudget
			out = append(out, Result{
				RuleID: "route-initial-js",
				Level:  lvl,
				Route:  r.Path,
				Detail: humanBytes(r.ThirdPartyJS) + " third-party JS (over by " + humanBytes(diff) + ")",
			})
		}
	}
	return out
}

// UnoptimizedImage flags <img> without width/height or missing loading="lazy".
func UnoptimizedImage(filePath, level string) []Result {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	s := string(b)
	imgTag := regexp.MustCompile(`(?is)<img\b[^>]*>`) // coarse match
	widthAttr := regexp.MustCompile(`\bwidth\s*=\s*"?\d+`)
	heightAttr := regexp.MustCompile(`\bheight\s*=\s*"?\d+`)
	loadingLazy := regexp.MustCompile(`\bloading\s*=\s*"?lazy"?`)
	var out []Result
	tags := imgTag.FindAllString(s, -1)
	for _, t := range tags {
		missingWH := !(widthAttr.MatchString(t) && heightAttr.MatchString(t))
		missingLazy := !loadingLazy.MatchString(t)
		if missingWH || missingLazy {
			var reasons []string
			if missingWH {
				reasons = append(reasons, "missing width/height")
			}
			if missingLazy {
				reasons = append(reasons, "missing loading=\"lazy\"")
			}
			out = append(out, Result{
				RuleID: "unoptimized-image",
				Level:  level,
				File:   filePath,
				Detail: "<img> " + strings.Join(reasons, ", ") + " in " + filePath,
			})
		}
	}
	return out
}

// FontDisplay flags @font-face without font-display: swap.
func FontDisplay(filePath, level string) []Result {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	s := string(b)
	// Very lightweight pattern for @font-face blocks
	block := regexp.MustCompile(`(?is)@font-face\s*\{[^}]*\}`)
	hasDisplay := regexp.MustCompile(`(?is)font-display\s*:\s*(swap|optional|fallback)\b`)
	var out []Result
	for _, m := range block.FindAllString(s, -1) {
		if !hasDisplay.MatchString(m) {
			out = append(out, Result{
				RuleID: "font-display",
				Level:  level,
				File:   filePath,
				Detail: "@font-face missing font-display in " + filePath,
			})
		}
	}
	return out
}

// Helpers
func parseSize(s string) int {
	if s == "" {
		return 0
	}
	lower := strings.ToLower(strings.TrimSpace(s))
	mult := 1
	switch {
	case strings.HasSuffix(lower, "kb"):
		mult = 1024
		lower = strings.TrimSuffix(lower, "kb")
	case strings.HasSuffix(lower, "kib"):
		mult = 1024
		lower = strings.TrimSuffix(lower, "kib")
	case strings.HasSuffix(lower, "b"):
		lower = strings.TrimSuffix(lower, "b")
	}
	var n int
	for _, ch := range lower {
		if ch >= '0' && ch <= '9' {
			n = n*10 + int(ch-'0')
		}
	}
	return n * mult
}

func humanBytes(n int) string {
	if n >= 1024 {
		kb := float64(n) / 1024.0
		return strings.TrimRight(strings.TrimRight(sprintfFloat(kb, 1), "0"), ".") + "KB"
	}
	return sprintfFloat(float64(n), 0) + "B"
}

func sprintfFloat(f float64, prec int) string {
	return floatToString(f, prec)
}
