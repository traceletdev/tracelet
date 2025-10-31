package stats

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"

	"tracelet/internal/config"
)

type RouteStat struct {
	Path         string `json:"path"`
	JSGzipBytes  int    `json:"jsGzipBytes"`
	ThirdPartyJS int    `json:"thirdPartyJsBytes,omitempty"`
}

type Stats struct {
	Routes []RouteStat `json:"routes"`
}

// FromFileOrInfer loads stats from a known path or returns empty stub to allow rules that don't depend on it.
func FromFileOrInfer(c config.Collect) Stats {
	candidate := c.StatsFile
	if candidate == "" {
		candidate = ".tracelet/stats.json"
	}
	b, err := os.ReadFile(candidate)
	if err == nil {
		var s Stats
		if json.Unmarshal(b, &s) == nil {
			return s
		}
	}
	// Try framework-specific defaults
	switch c.Framework {
	case "next":
		// Attempt to infer simple homepage size from Next manifests (best-effort)
		if s, ok := loadNextStats(); ok {
			return s
		}
	case "vite":
		if s, ok := loadViteStats(); ok {
			return s
		}
	}
	return Stats{Routes: []RouteStat{}}
}

func gzipSize(p string) int {
	b, err := os.ReadFile(p)
	if err != nil {
		return 0
	}
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, 6)
	if err != nil {
		return 0
	}
	if _, err := w.Write(b); err != nil {
		return 0
	}
	if err := w.Close(); err != nil {
		return 0
	}
	return buf.Len()
}

func loadNextStats() (Stats, bool) {
	// best-effort: read .next/build-manifest.json
	type buildManifest struct {
		Pages map[string][]string `json:"pages"`
	}
	bmPath := filepath.Join(".next", "build-manifest.json")
	b, err := os.ReadFile(bmPath)
	if err != nil {
		return Stats{}, false
	}
	var bm buildManifest
	if json.Unmarshal(b, &bm) != nil {
		return Stats{}, false
	}
	total := 0
	for _, chunk := range bm.Pages["/"] {
		p := filepath.Join(".next", chunk)
		total += gzipSize(p)
	}
	return Stats{Routes: []RouteStat{{Path: "/", JSGzipBytes: total}}}, true
}

func loadViteStats() (Stats, bool) {
	// best-effort: read dist/manifest.json and sum entry chunk sizes
	manifestPath := filepath.Join("dist", "manifest.json")
	b, err := os.ReadFile(manifestPath)
	if err != nil {
		return Stats{}, false
	}
	// minimal manifest structure
	var m map[string]struct {
		IsEntry bool   `json:"isEntry"`
		File    string `json:"file"`
	}
	if json.Unmarshal(b, &m) != nil {
		return Stats{}, false
	}
	total := 0
	for _, v := range m {
		if v.IsEntry {
			p := filepath.Join("dist", v.File)
			total += gzipSize(p)
		}
	}
	return Stats{Routes: []RouteStat{{Path: "/", JSGzipBytes: total}}}, true
}
