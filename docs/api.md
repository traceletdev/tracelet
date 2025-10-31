# API Reference

Tracelet provides both CLI and programmatic APIs.

## Installation

Install via npm:

```bash
npm install -D tracelet
```

Or use `npx` without installation:

```bash
npx tracelet <command>
```

## CLI Commands

All commands can be run via `npx tracelet <command>` or `tracelet <command>` if installed globally.

### `tracelet init`

Initialize a new Tracelet project.

```bash
npx tracelet init
```

Creates:
- `tracelet.config.json` - Default configuration
- `.tracelet/` - Directory for stats and baselines
- `.tracelet/.gitkeep` - Ensures directory is tracked

### `tracelet lint`

Lint routes against budgets and rules.

```bash
# Basic usage
npx tracelet lint

# JSON output for CI
npx tracelet lint --format json

# Lint specific file
npx tracelet lint --file src/components/Hero.tsx

# Custom config
npx tracelet lint --config custom-config.json

# CI mode (exit 1 on warnings)
npx tracelet lint --ci
```

**Flags:**
- `--format` (`table`|`json`) - Output format (default: `table`)
- `--scope` (`all`|`changed`) - Lint scope (default: `all`)
- `--file <path>` - Target file for focused lint
- `--config <path>` - Path to config file
- `--quiet` - Suppress non-critical logs
- `--ci` - CI mode (non-zero exit on warnings)

**Exit Codes:**
- `0` - Success (or warnings only in non-CI mode)
- `1` - Warnings present (CI mode only)
- `2` - Errors present

**Output Example:**
```
Route        JS(gzip)  Verdict
/            9KB       ✅
/product     47KB      ❌ over budget

Diagnostics:
❌ [route-initial-js] /product — 47KB JS (over by 12KB)
```

### `tracelet probe`

Probe a URL for performance metrics.

```bash
# Basic usage
npx tracelet probe http://localhost:3000

# Mobile profile
npx tracelet probe http://localhost:3000 --profile mobile

# Multiple runs (averaged)
npx tracelet probe http://localhost:3000 --runs 3

# Save to file
npx tracelet probe http://localhost:3000 --out .tracelet/probe.json

# Verbose output (show each run)
npx tracelet probe http://localhost:3000 --runs 3 --verbose
```

**Flags:**
- `--profile` (`desktop`|`mobile`) - Throttling profile (default: `desktop`)
- `--runs <number>` - Number of runs to average (default: 1)
- `--out <path>` - Write JSON to file
- `--verbose` - Log each run and include samples in JSON

**Metrics Collected:**
- `ttfb` - Time to First Byte (ms)
- `fcp` - First Contentful Paint (ms)
- `lcp` - Largest Contentful Paint (ms)
- `cls` - Cumulative Layout Shift (score)
- `tbtLite` - Total Blocking Time Lite (ms)
- `fsi` - First Script Idle (ms)

**JSON Output:**
```json
{
  "url": "http://localhost:3000",
  "profile": "desktop",
  "runAt": "2025-10-30T14:48:56Z",
  "metrics": {
    "ttfb": 342,
    "fcp": 564,
    "lcp": 804,
    "cls": 0,
    "tbtLite": 22,
    "fsi": 0
  },
  "runs": 3,
  "samples": [...]
}
```

### `tracelet ci`

Run CI checks with baseline comparison.

```bash
# Compare against baseline
npx tracelet ci --compare .tracelet/baseline.json

# Write new baseline
npx tracelet ci --write-baseline .tracelet/baseline.json

# Markdown output for PR comments
npx tracelet ci --compare .tracelet/baseline.json --format markdown
```

**Flags:**
- `--compare <path>` - Path to baseline JSON to compare against
- `--write-baseline <path>` - Write current results to baseline file
- `--format` (`markdown`|`json`) - Output format (default: `markdown`)
- `--config <path>` - Path to config file

**Exit Codes:**
- `0` - All checks passed
- `2` - Errors found

### `tracelet hud`

Start the HUD overlay server.

```bash
# Start on default port 3111
npx tracelet hud

# Custom port
npx tracelet hud --port 3001

# With config
npx tracelet hud --config tracelet.config.json
```

**Usage:**
1. Install tracelet: `npm install -D tracelet`
2. Start HUD server: `npx tracelet hud`
3. Inject script in your app: `<script src="http://localhost:3111/overlay.js"></script>`
4. See live performance feedback in browser

The HUD displays:
- Route statistics with budget status
- Real-time lint results
- Visual indicators (✓/⚠️/❌) for each route
- Collapsible interface

### `tracelet fix`

Apply automatic fixes (placeholder for v0.1.0).

```bash
npx tracelet fix

# Dry run (show what would be fixed)
npx tracelet fix

# Apply fixes
npx tracelet fix --apply
```

Currently a placeholder. Future versions will support automatic fixes for:
- Adding `loading="lazy"` to images
- Adding `font-display: swap` to fonts

## Programmatic API

### Go Packages

#### `internal/config`

Load and validate configuration.

```go
import "tracelet/internal/config"

cfg, err := config.Load("tracelet.config.json")
if err != nil {
    log.Fatal(err)
}
```

#### `internal/lint`

Run linting engine.

```go
import "tracelet/internal/lint"

req := lint.Request{
    Scope:  "all",
    Config: cfg,
}
results, stats := lint.Run(req)
```

#### `internal/probe`

Run performance probe.

```go
import "tracelet/internal/probe"

req := probe.Request{
    URL:     "http://localhost:3000",
    Profile: probe.ProfileDesktop,
    Runs:    3,
}
resp := probe.Run(req)
```

#### `internal/reporters`

Format and output results.

```go
import "tracelet/internal/reporters"

// Table output
reporters.PrintTable(os.Stdout, results, stats)

// JSON output
reporters.PrintJSON(os.Stdout, results, stats)

// Markdown output (for PR comments)
reporters.PrintMarkdown(os.Stdout, results, stats, baseline, diff)
```

## Stats Schema

### `.tracelet/stats.json`

Generated by adapters:

```json
{
  "routes": [
    {
      "path": "/",
      "jsGzipBytes": 12288,
      "thirdPartyJsBytes": 20480
    },
    {
      "path": "/product",
      "jsGzipBytes": 40960
    }
  ]
}
```

### Baseline Format

```json
{
  "results": [...],
  "stats": {
    "routes": [...]
  }
}
```

## Environment Variables

- `CHROME_PATH` - Custom path to Chrome/Chromium binary for probe
- `TRACELET_CONFIG` - Default config path (if not in current directory)

## GitHub Action

See `.github/workflows/tracelet.yml` for example usage.

```yaml
- uses: actions/setup-go@v5
- name: Run Tracelet
  run: |
    go build -o tracelet ./cmd/tracelet
    ./tracelet ci --compare .tracelet/baseline.json
```

## VS Code Extension API

See `ui/vscode-extension/README.md` for extension development.

Extension exposes:
- `tracelet.lintChanged` - Command to lint current file
- `tracelet.probeCurrentRoute` - Command to probe a URL
- `tracelet.openConfig` - Command to open config file

Settings:
- `tracelet.binaryPath` - Path to tracelet binary
- `tracelet.debounceMs` - Lint debounce delay
- `tracelet.enableOnType` - Enable linting on type
- `tracelet.probe.profile` - Default probe profile

