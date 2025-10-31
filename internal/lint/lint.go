package lint

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"tracelet/internal/config"
	"tracelet/internal/lint/rules"
	"tracelet/internal/stats"
)

type Request struct {
	Scope  string
	File   string
	Config config.Config
}

type Results []rules.Result

func (r Results) HasLevel(level string) bool {
	for _, x := range r {
		if strings.EqualFold(x.Level, level) {
			return true
		}
	}
	return false
}

func Run(req Request) (Results, stats.Stats) {
	// Load stats from adapter output
	st := stats.FromFileOrInfer(req.Config.Collect)

	var res Results

	// route-initial-js
	if lvl := req.Config.Rules["route-initial-js"]; lvl != "off" {
		res = append(res, rules.RouteInitialJS(req.Config, st)...)
	}

	// unoptimized-image
	if lvl := req.Config.Rules["unoptimized-image"]; lvl != "off" {
		filesToCheck := []string{}
		if req.File != "" {
			filesToCheck = []string{req.File}
		} else if req.Scope == "changed" {
			filesToCheck = getChangedFiles([]string{".html", ".tsx", ".jsx"})
		} else if req.Scope == "all" {
			filesToCheck = findFiles([]string{".html", ".tsx", ".jsx"})
		}
		for _, file := range filesToCheck {
			if strings.HasSuffix(strings.ToLower(file), ".html") || strings.HasSuffix(strings.ToLower(file), ".tsx") || strings.HasSuffix(strings.ToLower(file), ".jsx") {
				res = append(res, rules.UnoptimizedImage(file, lvl)...)
			}
		}
	}

	// font-display
	if lvl := req.Config.Rules["font-display"]; lvl != "off" {
		filesToCheck := []string{}
		if req.File != "" {
			filesToCheck = []string{req.File}
		} else if req.Scope == "changed" {
			filesToCheck = getChangedFiles([]string{".css", ".scss"})
		} else if req.Scope == "all" {
			filesToCheck = findFiles([]string{".css", ".scss"})
		}
		for _, file := range filesToCheck {
			if filepath.Ext(strings.ToLower(file)) == ".css" || strings.HasSuffix(strings.ToLower(file), ".scss") {
				res = append(res, rules.FontDisplay(file, lvl)...)
			}
		}
	}

	// missing-image-dimensions
	if lvl := req.Config.Rules["missing-image-dimensions"]; lvl != "off" {
		filesToCheck := []string{}
		if req.File != "" {
			filesToCheck = []string{req.File}
		} else if req.Scope == "changed" {
			filesToCheck = getChangedFiles([]string{".html", ".tsx", ".jsx"})
		} else if req.Scope == "all" {
			filesToCheck = findFiles([]string{".html", ".tsx", ".jsx"})
		}
		for _, file := range filesToCheck {
			if strings.HasSuffix(strings.ToLower(file), ".html") || strings.HasSuffix(strings.ToLower(file), ".tsx") || strings.HasSuffix(strings.ToLower(file), ".jsx") {
				res = append(res, rules.MissingImageDimensions(file, lvl)...)
			}
		}
	}

	// render-blocking-resources
	if lvl := req.Config.Rules["render-blocking-resources"]; lvl != "off" {
		filesToCheck := []string{}
		if req.File != "" {
			filesToCheck = []string{req.File}
		} else if req.Scope == "changed" {
			filesToCheck = getChangedFiles([]string{".html", ".tsx", ".jsx"})
		} else if req.Scope == "all" {
			filesToCheck = findFiles([]string{".html", ".tsx", ".jsx"})
		}
		for _, file := range filesToCheck {
			if strings.HasSuffix(strings.ToLower(file), ".html") || strings.HasSuffix(strings.ToLower(file), ".tsx") || strings.HasSuffix(strings.ToLower(file), ".jsx") {
				res = append(res, rules.RenderBlockingResources(file, lvl)...)
			}
		}
	}

	// missing-preconnect
	if lvl := req.Config.Rules["missing-preconnect"]; lvl != "off" {
		filesToCheck := []string{}
		if req.File != "" {
			filesToCheck = []string{req.File}
		} else if req.Scope == "changed" {
			filesToCheck = getChangedFiles([]string{".html", ".tsx", ".jsx"})
		} else if req.Scope == "all" {
			filesToCheck = findFiles([]string{".html", ".tsx", ".jsx"})
		}
		for _, file := range filesToCheck {
			if strings.HasSuffix(strings.ToLower(file), ".html") || strings.HasSuffix(strings.ToLower(file), ".tsx") || strings.HasSuffix(strings.ToLower(file), ".jsx") {
				res = append(res, rules.MissingPreconnect(file, lvl)...)
			}
		}
	}

	// missing-alt-text
	if lvl := req.Config.Rules["missing-alt-text"]; lvl != "off" {
		filesToCheck := []string{}
		if req.File != "" {
			filesToCheck = []string{req.File}
		} else if req.Scope == "changed" {
			filesToCheck = getChangedFiles([]string{".html", ".tsx", ".jsx"})
		} else if req.Scope == "all" {
			filesToCheck = findFiles([]string{".html", ".tsx", ".jsx"})
		}
		for _, file := range filesToCheck {
			if strings.HasSuffix(strings.ToLower(file), ".html") || strings.HasSuffix(strings.ToLower(file), ".tsx") || strings.HasSuffix(strings.ToLower(file), ".jsx") {
				res = append(res, rules.MissingAltText(file, lvl)...)
			}
		}
	}

	// react-inline-props
	if lvl := req.Config.Rules["react-inline-props"]; lvl != "off" {
		filesToCheck := []string{}
		if req.File != "" {
			filesToCheck = []string{req.File}
		} else if req.Scope == "changed" {
			filesToCheck = getChangedFiles([]string{".tsx", ".jsx"})
		} else if req.Scope == "all" {
			filesToCheck = findFiles([]string{".tsx", ".jsx"})
		}
		for _, file := range filesToCheck {
			if strings.HasSuffix(strings.ToLower(file), ".tsx") || strings.HasSuffix(strings.ToLower(file), ".jsx") {
				res = append(res, rules.ReactInlineProps(file, lvl)...)
			}
		}
	}

	// react-missing-memo
	if lvl := req.Config.Rules["react-missing-memo"]; lvl != "off" {
		filesToCheck := []string{}
		if req.File != "" {
			filesToCheck = []string{req.File}
		} else if req.Scope == "changed" {
			filesToCheck = getChangedFiles([]string{".tsx", ".jsx"})
		} else if req.Scope == "all" {
			filesToCheck = findFiles([]string{".tsx", ".jsx"})
		}
		for _, file := range filesToCheck {
			if strings.HasSuffix(strings.ToLower(file), ".tsx") || strings.HasSuffix(strings.ToLower(file), ".jsx") {
				res = append(res, rules.ReactMissingMemo(file, lvl)...)
			}
		}
	}

	// react-unstable-props
	if lvl := req.Config.Rules["react-unstable-props"]; lvl != "off" {
		filesToCheck := []string{}
		if req.File != "" {
			filesToCheck = []string{req.File}
		} else if req.Scope == "changed" {
			filesToCheck = getChangedFiles([]string{".tsx", ".jsx"})
		} else if req.Scope == "all" {
			filesToCheck = findFiles([]string{".tsx", ".jsx"})
		}
		for _, file := range filesToCheck {
			if strings.HasSuffix(strings.ToLower(file), ".tsx") || strings.HasSuffix(strings.ToLower(file), ".jsx") {
				res = append(res, rules.ReactUnstableProps(file, lvl)...)
			}
		}
	}

	// Normalize levels based on config where needed
	for i := range res {
		if res[i].Level == "" {
			// default to configured level
			res[i].Level = levelFor(res[i].RuleID, req.Config)
		}
	}
	return res, st
}

func levelFor(ruleID string, cfg config.Config) string {
	if lvl, ok := cfg.Rules[ruleID]; ok && lvl != "" {
		return lvl
	}
	return "info"
}

func getChangedFiles(extensions []string) []string {
	// Get both staged and unstaged changes
	// Unstaged changes
	cmd1 := exec.Command("git", "diff", "--name-only", "--diff-filter=ACMR")
	out1, _ := cmd1.Output()
	// Staged changes
	cmd2 := exec.Command("git", "diff", "--cached", "--name-only", "--diff-filter=ACMR")
	out2, _ := cmd2.Output()

	// Combine and deduplicate
	allChanged := make(map[string]bool)
	for _, f := range strings.Split(strings.TrimSpace(string(out1)), "\n") {
		if f != "" {
			allChanged[f] = true
		}
	}
	for _, f := range strings.Split(strings.TrimSpace(string(out2)), "\n") {
		if f != "" {
			allChanged[f] = true
		}
	}

	if len(allChanged) == 0 {
		return []string{}
	}

	var files []string
	for f := range allChanged {
		ext := filepath.Ext(strings.ToLower(f))
		for _, wantExt := range extensions {
			if ext == wantExt || strings.HasSuffix(strings.ToLower(f), wantExt) {
				// Resolve to absolute path
				if abs, err := filepath.Abs(f); err == nil {
					files = append(files, abs)
				} else {
					files = append(files, f)
				}
				break
			}
		}
	}
	return files
}

func findFiles(extensions []string) []string {
	var files []string
	skipDirs := map[string]bool{
		"node_modules": true,
		".next":        true,
		".git":         true,
		"dist":         true,
		"build":        true,
	}

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(strings.ToLower(path))
		for _, wantExt := range extensions {
			if ext == wantExt || strings.HasSuffix(strings.ToLower(path), wantExt) {
				files = append(files, path)
				break
			}
		}
		return nil
	})
	if err != nil {
		return files // Return what we found
	}
	return files
}
