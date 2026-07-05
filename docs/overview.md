# Tracelet Overview

**Tracelet** is a modern performance toolkit that brings **Lighthouse-level insight** and **ESLint-level ergonomics** into one seamless workflow.

It keeps performance **visible**, **linted**, and **actionable** from development to CI to production — without ever leaving your editor.

> **Tagline:** *Small traces. Big clarity.*

## Core Philosophy

| Principle | Description |
|-----------|-------------|
| **Speed** | Every operation completes in seconds — never minutes. |
| **Determinism** | Metrics should be reproducible regardless of environment. |
| **Integration** | Fits naturally into existing dev loops, editors, and CI. |
| **Transparency** | Exposes what's happening instead of abstracting it away. |

## Quick Start

### Installation

**Recommended: Install via npm**

```bash
# Install tracelet CLI
npm install -D @traceletdev/cli

# Install framework plugin (choose one)
npm install -D @traceletdev/next    # for Next.js
npm install -D @traceletdev/vite    # for Vite
```

**Or build from source:**

```bash
go build -o tracelet ./cmd/tracelet
```

### Initialize

```bash
# With npm
npx tracelet init

# Or if built from source
./tracelet init
```

This creates `tracelet.config.json` and `.tracelet/` directory.

### Framework Setup

**Next.js:**

```json
// package.json
{
  "scripts": {
    "postbuild": "node node_modules/@traceletdev/next/collect.js"
  }
}
```

**Vite:**

```js
// vite.config.js
import tracelet from '@traceletdev/vite';
export default defineConfig({ plugins: [tracelet()] });
```

### Run Lint

```bash
# Lint your routes
npx tracelet lint

# JSON output for CI
npx tracelet lint --format json
```

### Probe a Route

```bash
# Probe local development server
npx tracelet probe http://localhost:3000

# Mobile profile
npx tracelet probe http://localhost:3000 --profile mobile --runs 3
```

## Features

### 1. **Tracelet Lint**

- Lints per-route budgets (`initialJs`, `thirdPartyJs`, etc.)
- Rules for `unoptimized-image`, `font-display`, `large-lib`
- Supports Next.js, Vite, Astro (via adapters)
- Autofix for HTML/CSS-based optimizations
- JSON, CLI, and Markdown outputs

### 2. **Tracelet Probe**

- Fast, deterministic Chrome-based performance audit
- Collects: **TTFB, FCP, LCP, CLS, TBT-Lite, FSI**
- Runs in 2–3 seconds
- `--profile` options (`desktop`, `mobile`)
- Outputs JSON for CI and HUD consumption

### 3. **Tracelet HUD**

- Real-time overlay for development servers
- Connects via WebSocket
- Displays route budgets + live probe snapshots
- Visual color-coding for over-budget areas

### 4. **Tracelet CI**

- Runs `lint` and `probe` in pipelines
- Compares results with baselines
- Posts GitHub PR comments summarizing changes
- Fails builds if "error"-level rules are violated

### 5. **VS Code Integration**

- In-editor diagnostics with squiggles
- Quick Fix actions for common issues
- Status bar showing route sizes
- Probe command from Command Palette

## User Journeys

### Developer (Local)

1. Install: `npm i -D @traceletdev/cli && tracelet init`
2. Run: `tracelet lint`
3. Fix issues via `tracelet fix` or in-editor Quick Fix
4. See live feedback via HUD overlay

### Team (CI/CD)

1. Add GitHub Action to `.github/workflows/tracelet.yml`
2. On PR: Bot posts Markdown table comparing metrics
3. CI fails if "error"-level thresholds breached

### Tech Lead

- Run `tracelet ci --compare=baseline` for trend analysis
- Monitor performance evolution via baseline diffs

## Next Steps

- [Configuration Guide](./config.md)
- [Rules Reference](./rules.md)
- [API Documentation](./api.md)
- [Adapter Setup](../adapters/)

## Philosophy

Tracelet isn't another framework or metric viewer —
it's the **performance layer of your development loop**.

It's built for speed, clarity, and integration:

- **Lint** performance budgets like code
- **Probe** runtime behavior in seconds
- **See** results live in your editor or CI

> **Tracelet** — performance you can lint.
