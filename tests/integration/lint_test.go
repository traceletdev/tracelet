package integration

import (
    "os"
    "path/filepath"
    "runtime"
    "testing"

    "tracelet/internal/config"
    "tracelet/internal/lint"
)

func getFixtureDir(name string) string {
    _, filename, _, _ := runtime.Caller(1) // Skip getFixtureDir, get test function's file
    testDir := filepath.Dir(filename)
    repoRoot := filepath.Join(testDir, "..", "..")
    return filepath.Join(repoRoot, "tests", "fixtures", name)
}

func TestNextBasic_LintOverBudget(t *testing.T) {
    fixtureDir := getFixtureDir("next-basic")
    cwd, _ := os.Getwd()
    if err := os.Chdir(fixtureDir); err != nil {
        t.Fatalf("chdir: %v", err)
    }
    t.Cleanup(func() { _ = os.Chdir(cwd) })
    cfg, err := config.Load("tracelet.config.json")
    if err != nil {
        t.Fatalf("load config: %v", err)
    }
    results, _ := lint.Run(lint.Request{Scope: "all", Config: cfg})
    if !results.HasLevel("error") {
        t.Fatalf("expected at least one error due to over budget, got none")
    }
}

func TestViteBasic_LintOverBudget(t *testing.T) {
    fixtureDir := getFixtureDir("vite-basic")
    cwd, _ := os.Getwd()
    if err := os.Chdir(fixtureDir); err != nil {
        t.Fatalf("chdir: %v", err)
    }
    t.Cleanup(func() { _ = os.Chdir(cwd) })
    cfg, err := config.Load("tracelet.config.json")
    if err != nil {
        t.Fatalf("load config: %v", err)
    }
    results, _ := lint.Run(lint.Request{Scope: "all", Config: cfg})
    if !results.HasLevel("error") {
        t.Fatalf("expected at least one error due to over budget, got none")
    }
}


