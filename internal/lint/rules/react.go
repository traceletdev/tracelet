package rules

import (
	"os"
	"regexp"
	"strings"
)

// ReactInlineProps flags inline function/object literals in JSX props that can cause unnecessary rerenders.
func ReactInlineProps(filePath, level string) []Result {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	s := string(b)

	// Only check .tsx and .jsx files
	if !strings.HasSuffix(strings.ToLower(filePath), ".tsx") && !strings.HasSuffix(strings.ToLower(filePath), ".jsx") {
		return nil
	}

	// Check for React imports to confirm this is a React file
	hasReactImport := regexp.MustCompile(`(?m)^import\s+.*\bfrom\s+['"]react['"]`).MatchString(s)
	if !hasReactImport && !strings.Contains(s, "React.") {
		// Might still be React if using JSX pragma, check for JSX syntax
		if !regexp.MustCompile(`<[A-Z][a-zA-Z0-9]*\s`).MatchString(s) {
			return nil
		}
	}

	var out []Result

	// Pattern 1: Inline arrow functions in props: onClick={() => {...}}
	// Matches: propName={(() => { ... }) or propName={() => ... }
	inlineArrowFunc := regexp.MustCompile(`(\w+)\s*=\s*\{\s*\([^)]*\)\s*=>\s*`)
	matches := inlineArrowFunc.FindAllStringSubmatch(s, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			propName := match[1]
			// Skip if it's already useCallback or useMemo
			context := extractContext(s, match[0])
			if !strings.Contains(context, "useCallback") && !strings.Contains(context, "useMemo") {
				out = append(out, Result{
					RuleID: "react-inline-props",
					Level:  level,
					File:   filePath,
					Detail: "Inline arrow function in prop '" + propName + "' can cause unnecessary rerenders. Consider using useCallback.",
				})
			}
		}
	}

	// Pattern 2: Inline object literals in props: style={{...}} or data={{...}}
	// Matches: propName={{ ... }} where the object is created inline
	inlineObject := regexp.MustCompile(`(\w+)\s*=\s*\{\s*\{`)
	objMatches := inlineObject.FindAllStringSubmatch(s, -1)
	for _, match := range objMatches {
		if len(match) >= 2 {
			propName := match[1]
			// Skip style prop if it's a common pattern (but still flag others)
			if propName == "style" {
				// Check if it's a simple style object (might be acceptable)
				context := extractContext(s, match[0])
				// If it has multiple properties or complex structure, flag it
				if strings.Count(context, ":") > 1 {
					out = append(out, Result{
						RuleID: "react-inline-props",
						Level:  level,
						File:   filePath,
						Detail: "Inline object literal in prop '" + propName + "' can cause unnecessary rerenders. Consider moving it outside the render or using useMemo.",
					})
				}
			} else {
				out = append(out, Result{
					RuleID: "react-inline-props",
					Level:  level,
					File:   filePath,
					Detail: "Inline object literal in prop '" + propName + "' can cause unnecessary rerenders. Consider moving it outside the render or using useMemo.",
				})
			}
		}
	}

	return out
}

// ReactMissingMemo flags components that receive props but aren't memoized.
func ReactMissingMemo(filePath, level string) []Result {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	s := string(b)

	// Only check .tsx and .jsx files
	if !strings.HasSuffix(strings.ToLower(filePath), ".tsx") && !strings.HasSuffix(strings.ToLower(filePath), ".jsx") {
		return nil
	}

	// Check for React imports
	hasReactImport := regexp.MustCompile(`(?m)^import\s+.*\bfrom\s+['"]react['"]`).MatchString(s)
	if !hasReactImport && !strings.Contains(s, "React.") {
		if !regexp.MustCompile(`<[A-Z][a-zA-Z0-9]*\s`).MatchString(s) {
			return nil
		}
	}

	var out []Result

	// Find function components that receive props but aren't memoized
	// Pattern: function ComponentName(props) or const ComponentName = (props) =>
	funcComponent := regexp.MustCompile(`(?:function|const)\s+([A-Z][a-zA-Z0-9]*)\s*(?:=|\([^)]*\)\s*=>|\([^)]*\{)`)
	matches := funcComponent.FindAllStringSubmatch(s, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		componentName := match[1]

		// Check if component receives props
		hasProps := strings.Contains(match[0], "props") || strings.Contains(match[0], "{")

		// Check if already memoized
		// Look for React.memo(ComponentName) or export default React.memo(...)
		memoPattern := regexp.MustCompile(`(?:React\.)?memo\s*\(\s*` + regexp.QuoteMeta(componentName))
		if memoPattern.MatchString(s) {
			continue
		}

		// Check if component is exported with memo
		exportMemoPattern := regexp.MustCompile(`export\s+(?:default\s+)?(?:React\.)?memo`)
		if exportMemoPattern.MatchString(s) {
			continue
		}

		if hasProps {
			// Check if it's a hook or utility function (usually lowercase or specific patterns)
			if strings.HasPrefix(componentName, "use") || strings.ToLower(componentName) == componentName {
				continue
			}

			out = append(out, Result{
				RuleID: "react-missing-memo",
				Level:  level,
				File:   filePath,
				Detail: "Component '" + componentName + "' receives props but isn't memoized. Consider wrapping with React.memo if it renders frequently.",
			})
		}
	}

	return out
}

// ReactUnstableProps flags props that are object/function literals passed from parent components.
func ReactUnstableProps(filePath, level string) []Result {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	s := string(b)

	// Only check .tsx and .jsx files
	if !strings.HasSuffix(strings.ToLower(filePath), ".tsx") && !strings.HasSuffix(strings.ToLower(filePath), ".jsx") {
		return nil
	}

	// Check for React imports
	hasReactImport := regexp.MustCompile(`(?m)^import\s+.*\bfrom\s+['"]react['"]`).MatchString(s)
	if !hasReactImport && !strings.Contains(s, "React.") {
		if !regexp.MustCompile(`<[A-Z][a-zA-Z0-9]*\s`).MatchString(s) {
			return nil
		}
	}

	var out []Result

	// Find JSX elements where props contain inline functions or objects
	// Pattern: <ComponentName propName={() => ...} or propName={{...}} />
	jsxWithInlineProps := regexp.MustCompile(`<([A-Z][a-zA-Z0-9]*)\s+([^>]*=\s*\{[^}]*\}[^>]*)>?`)
	matches := jsxWithInlineProps.FindAllStringSubmatch(s, -1)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		componentName := match[1]
		propsStr := match[2]

		// Check for inline arrow functions
		if regexp.MustCompile(`\w+\s*=\s*\{\s*\([^)]*\)\s*=>`).MatchString(propsStr) {
			// Extract prop name
			propMatch := regexp.MustCompile(`(\w+)\s*=\s*\{\s*\([^)]*\)\s*=>`)
			if propMatch := propMatch.FindStringSubmatch(propsStr); len(propMatch) >= 2 {
				propName := propMatch[1]
				out = append(out, Result{
					RuleID: "react-unstable-props",
					Level:  level,
					File:   filePath,
					Detail: "Component '" + componentName + "' receives unstable prop '" + propName + "' (inline function). This will cause rerenders. Move function outside render or use useCallback in parent.",
				})
			}
		}

		// Check for inline object literals (non-style)
		if regexp.MustCompile(`(\w+)\s*=\s*\{\s*\{`).MatchString(propsStr) {
			objMatch := regexp.MustCompile(`(\w+)\s*=\s*\{\s*\{`)
			if objMatch := objMatch.FindStringSubmatch(propsStr); len(objMatch) >= 2 {
				propName := objMatch[1]
				if propName != "style" {
					out = append(out, Result{
						RuleID: "react-unstable-props",
						Level:  level,
						File:   filePath,
						Detail: "Component '" + componentName + "' receives unstable prop '" + propName + "' (inline object). This will cause rerenders. Move object outside render or use useMemo in parent.",
					})
				}
			}
		}
	}

	return out
}

// extractContext extracts a snippet of code around a match for context checking
func extractContext(s, match string) string {
	idx := strings.Index(s, match)
	if idx == -1 {
		return ""
	}
	start := idx - 100
	if start < 0 {
		start = 0
	}
	end := idx + len(match) + 100
	if end > len(s) {
		end = len(s)
	}
	return s[start:end]
}
