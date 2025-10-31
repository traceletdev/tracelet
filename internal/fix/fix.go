package fix

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

type Fix struct {
	RuleID  string
	File    string
	Applied bool
	Error   error
}

func ApplyImageFix(filePath string) (Fix, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return Fix{RuleID: "unoptimized-image", File: filePath}, err
	}

	content := string(b)
	modified := false

	// Find all img tags and fix them
	imgPattern := regexp.MustCompile(`<img\b([^>]*)>`)
	content = imgPattern.ReplaceAllStringFunc(content, func(match string) string {
		if strings.Contains(match, "loading=") {
			return match // Already has loading
		}
		modified = true
		// Insert loading="lazy" before closing >
		return strings.Replace(match, ">", ` loading="lazy">`, 1)
	})

	if !modified {
		return Fix{RuleID: "unoptimized-image", File: filePath, Applied: false}, nil
	}

	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return Fix{RuleID: "unoptimized-image", File: filePath}, err
	}

	return Fix{RuleID: "unoptimized-image", File: filePath, Applied: true}, nil
}

func ApplyFontDisplayFix(filePath string) (Fix, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return Fix{RuleID: "font-display", File: filePath}, err
	}

	content := string(b)
	modified := false

	// Find @font-face blocks missing font-display
	fontFacePattern := regexp.MustCompile(`(@font-face\s*\{[^}]*)(\})`)
	content = fontFacePattern.ReplaceAllStringFunc(content, func(match string) string {
		if strings.Contains(match, "font-display") {
			return match // Already has font-display
		}
		modified = true
		// Add font-display: swap; before closing brace
		return strings.Replace(match, "}", "  font-display: swap;\n}", 1)
	})

	if !modified {
		return Fix{RuleID: "font-display", File: filePath, Applied: false}, nil
	}

	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return Fix{RuleID: "font-display", File: filePath}, err
	}

	return Fix{RuleID: "font-display", File: filePath, Applied: true}, nil
}

func ApplyRenderBlockingFix(filePath string) (Fix, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return Fix{RuleID: "render-blocking-resources", File: filePath}, err
	}

	content := string(b)
	modified := false

	// Find <head> section
	headMatch := regexp.MustCompile(`(?is)(<head[^>]*>)(.*?)(</head>)`)
	content = headMatch.ReplaceAllStringFunc(content, func(match string) string {
		parts := headMatch.FindStringSubmatch(match)
		if len(parts) < 4 {
			return match
		}
		headOpen := parts[1]
		headContent := parts[2]
		headClose := parts[3]

		// Add defer to scripts without defer/async/module
		scriptPattern := regexp.MustCompile(`(?is)(<script\b([^>]*)>)`)
		hasDeferAsync := regexp.MustCompile(`\b(defer|async|type\s*=\s*"?module"?)`)
		headContent = scriptPattern.ReplaceAllStringFunc(headContent, func(scriptMatch string) string {
			scriptParts := scriptPattern.FindStringSubmatch(scriptMatch)
			if len(scriptParts) >= 3 && !hasDeferAsync.MatchString(scriptParts[2]) {
				modified = true
				// Add defer before closing >
				return strings.Replace(scriptMatch, ">", ` defer>`, 1)
			}
			return scriptMatch
		})

		return headOpen + headContent + headClose
	})

	if !modified {
		return Fix{RuleID: "render-blocking-resources", File: filePath, Applied: false}, nil
	}

	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return Fix{RuleID: "render-blocking-resources", File: filePath}, err
	}

	return Fix{RuleID: "render-blocking-resources", File: filePath, Applied: true}, nil
}

func ApplyAltTextFix(filePath string) (Fix, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return Fix{RuleID: "missing-alt-text", File: filePath}, err
	}

	content := string(b)
	modified := false

	// Find all img tags and add alt="" if missing
	imgPattern := regexp.MustCompile(`<img\b([^>]*)>`)
	altAttr := regexp.MustCompile(`\balt\s*=`)
	content = imgPattern.ReplaceAllStringFunc(content, func(match string) string {
		parts := imgPattern.FindStringSubmatch(match)
		if len(parts) >= 2 && !altAttr.MatchString(parts[1]) {
			modified = true
			// Insert alt="" before closing >
			return strings.Replace(match, ">", ` alt="">`, 1)
		}
		return match
	})

	if !modified {
		return Fix{RuleID: "missing-alt-text", File: filePath, Applied: false}, nil
	}

	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return Fix{RuleID: "missing-alt-text", File: filePath}, err
	}

	return Fix{RuleID: "missing-alt-text", File: filePath, Applied: true}, nil
}

func ApplyFix(ruleID, filePath string) (Fix, error) {
	switch ruleID {
	case "unoptimized-image":
		return ApplyImageFix(filePath)
	case "font-display":
		return ApplyFontDisplayFix(filePath)
	case "render-blocking-resources":
		return ApplyRenderBlockingFix(filePath)
	case "missing-alt-text":
		return ApplyAltTextFix(filePath)
	default:
		return Fix{RuleID: ruleID, File: filePath}, fmt.Errorf("no fix available for rule: %s", ruleID)
	}
}

func FormatFixResult(f Fix) string {
	if f.Error != nil {
		return fmt.Sprintf("❌ %s (%s): %v", f.RuleID, f.File, f.Error)
	}
	if f.Applied {
		return fmt.Sprintf("✅ Fixed %s in %s", f.RuleID, f.File)
	}
	return fmt.Sprintf("ℹ️  No fix needed for %s in %s", f.RuleID, f.File)
}
