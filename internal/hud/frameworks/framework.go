package frameworks

import _ "embed"

//go:embed react.js
var reactInstrumentationJS string

// Component represents a React/component framework component with render tracking.
// Counts are aggregated by display name: RenderCount is the total across all
// instances, Instances is how many instances share the name, and MaxRenders is
// the hottest single instance (the signal that matters for re-render smells).
type Component struct {
	Name        string                 `json:"name"`
	RenderCount int                    `json:"renderCount"`
	Instances   int                    `json:"instances,omitempty"`
	MaxRenders  int                    `json:"maxRenders,omitempty"`
	Props       map[string]interface{} `json:"props,omitempty"`
	Children    []*Component           `json:"children,omitempty"`
}

// Instrumentation represents a framework instrumentation instance
type Instrumentation interface {
	// Detect checks if this framework is present in the page
	Detect() bool

	// Install returns the JavaScript instrumentation code to inject into the page
	Install() string

	// FrameworkName returns the name of the framework (e.g., "react", "vue")
	FrameworkName() string
}

// ReactInstrumentation implements Instrumentation for React
type ReactInstrumentation struct{}

func (r *ReactInstrumentation) FrameworkName() string {
	return "react"
}

func (r *ReactInstrumentation) Detect() bool {
	// Detection happens client-side via JavaScript.
	return true
}

// Install returns the canonical instrumentation from react.js (the same code
// shipped as the tracelet-react npm package).
func (r *ReactInstrumentation) Install() string {
	return reactInstrumentationJS
}

// GetInstrumentation returns the appropriate instrumentation for the detected framework
func GetInstrumentation(frameworkName string) Instrumentation {
	switch frameworkName {
	case "react":
		return &ReactInstrumentation{}
	default:
		return nil
	}
}
