package hud

import (
	"strings"
	"testing"
)

// Guards the Shadow DOM isolation fix: the overlay must not leak its internal
// ids into the host page's DOM, and must render inside a shadow root.
func TestOverlayJSShadowDOMIsolation(t *testing.T) {
	js := overlayJS(3111)

	if !strings.Contains(js, "attachShadow") {
		t.Error("overlayJS() no longer attaches a shadow root; host-page DOM queries can see overlay internals again")
	}

	leakedIDs := []string{
		"__tracelet_hud", "__tracelet_toggle", "__tracelet_refresh_btn",
		"__tracelet_content", "__tracelet_routes", "__tracelet_metrics",
		"__tracelet_components_bar", "__tracelet_components",
	}
	for _, id := range leakedIDs {
		if strings.Contains(js, "'"+id+"'") {
			t.Errorf("overlayJS() assigns id %q, which a host page's document.querySelectorAll('[id]') can pick up", id)
		}
	}
}
