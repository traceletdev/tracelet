# 🧭 Tracelet — Planning Document

## Overview

**Tracelet** is a modern performance toolkit that brings **Lighthouse-level insight** and **ESLint-level ergonomics** into one seamless workflow.

It keeps performance **visible**, **linted**, and **actionable** from development to CI to production — without ever leaving your editor.

> **Tagline:** *Small traces. Big clarity.*

---

## 🔩 Core Philosophy

| Principle | Description |
|------------|--------------|
| **Speed** | Every operation completes in seconds — never minutes. |
| **Determinism** | Metrics should be reproducible regardless of environment. |
| **Integration** | Fits naturally into existing dev loops, editors, and CI. |
| **Transparency** | Exposes what’s happening instead of abstracting it away. |

---

## 🧱 Repository Structure

```
tracelet/
├─ README.md
├─ docs/
│  ├─ PLANNING.md
│  ├─ overview.md
│  ├─ config.md
│  ├─ rules.md
│  └─ api.md
├─ cmd/
│  └─ tracelet/               # Go CLI entrypoint
│     └─ main.go
├─ internal/
│  ├─ config/                 # load/validate config
│  ├─ lint/                   # static analysis + budgets
│  ├─ probe/                  # fast Chrome CDP metrics
│  ├─ hud/                    # websocket overlay server
│  ├─ reporters/              # cli/json/md reporters
│  ├─ ci/                     # GitHub action helpers
│  ├─ lsp/                    # Language Server for editor integration
│  └─ util/                   # gzip, file utils, error helpers
├─ adapters/
│  ├─ vite-plugin-tracelet.cjs
│  ├─ next-plugin-tracelet.js
│  └─ astro-plugin-tracelet.mjs
├─ ui/
│  ├─ vscode-extension/       # Editor integration
│  └─ overlay/                # Dev HUD client
├─ tests/
│  ├─ fixtures/
│  └─ e2e/
├─ tracelet.config.json
├─ go.mod / go.sum
└─ .github/
   └─ workflows/
      └─ tracelet.yml
```

---

## 🧩 Feature Set

### 1. **Tracelet Lint**

- Lints per-route budgets (`initialJs`, `thirdPartyJs`, etc.).
- Rules for `unoptimized-image`, `font-display`, `large-lib`.
- Supports Next.js, Vite, Astro (via adapters).
- Autofix for HTML/CSS-based optimizations.
- JSON, CLI, and Markdown outputs.

### 2. **Tracelet Probe**

- Fast, deterministic Chrome-based performance audit.
- Collects: **TTFB, FCP, LCP, CLS, TBT-Lite, FSI**.
- Runs in 2–3 seconds.
- `--profile` options (`desktop`, `mobile`).
- Outputs JSON for CI and HUD consumption.

### 3. **Tracelet HUD**

- Real-time overlay for development servers.
- Connects via WebSocket.
- Displays route budgets + live probe snapshots.
- Visual color-coding for over-budget areas.

### 4. **Tracelet CI**

- Runs `lint` and `probe` in pipelines.
- Compares results with baselines.
- Posts GitHub PR comments summarizing changes.
- Fails builds if “error”-level rules are violated.

### 5. **Tracelet Config**

Centralized config file `tracelet.config.json`.

```json
{
  "extends": ["recommended"],
  "budgets": {
    "default": { "initialJs": "35kb", "thirdPartyJs": "50kb" },
    "/": { "initialJs": "10kb" }
  },
  "rules": {
    "route-initial-js": "error",
    "unoptimized-image": "warn"
  },
  "probe": { "profile": "desktop", "runs": 1 },
  "collect": { "framework": "next", "statsFile": ".tracelet/stats.json" }
}
```

---

## 🧠 User Journeys

### 🧑‍💻 1. Developer (Local)

**Goal:** catch regressions instantly while coding.

1. Install:

   ```bash
   npm i -D tracelet
   tracelet init
   ```

2. Run:

   ```bash
   tracelet lint
   tracelet probe http://localhost:3000
   ```

3. Output:

   ```
   Route        JS(gzip)  FCP  LCP  CLS  Verdict
   /            9KB       980  1200 0.02 ✅
   /product     47KB      1350 2100 0.09 ⚠️ over budget
   ```

4. Fix issues via `tracelet fix` or the in-editor Quick Fix.

---

### ⚙️ 2. Team (CI/CD)

**Goal:** enforce performance budgets automatically.

1. Add GitHub Action:

   ```yaml
   - uses: tracelet/action@v1
     with:
       config: tracelet.config.json
   ```

2. On PR:
   - Bot posts Markdown table comparing metrics.
   - CI fails if “error”-level thresholds breached.

---

### 👁️ 3. Tech Lead / Perf Champion

**Goal:** monitor performance evolution.

- Run `tracelet ci --compare=baseline`.
- Weekly auto-summary posted to Slack.
- Optional Tracelet Cloud dashboard integration (future).

---

### 🧩 4. In-Editor Lint (ESLint-style)

**Goal:** show performance diagnostics inline while coding.

#### Flow

1. Install the **Tracelet VS Code extension**.
2. On file save (or after debounce), it runs:

   ```
   tracelet lint --format=json --scope=changed --file <current-file>
   ```

3. Warnings and errors appear in the **Problems panel**.
4. **Quick Fixes** available:
   - Add `loading="lazy"`
   - Add `font-display: swap`
   - Suggest dynamic import for large modules
5. Status bar shows route size + overage delta.
6. Run **“Tracelet: Probe Current Route”** from Command Palette for live metrics.

#### Example diagnostics

```
❌ [route-initial-js] /product — 3.1 KB over 35 KB limit
⚠️  [unoptimized-image] Missing width/height on <img>
ℹ️  [font-display] Consider font-display: swap
```

#### Extension settings

```json
{
  "tracelet.binaryPath": "",
  "tracelet.debounceMs": 600,
  "tracelet.enableOnType": false,
  "tracelet.probe.profile": "desktop"
}
```

#### Architecture

- **Phase 1:** Thin VS Code shell → calls CLI JSON.
- **Phase 2:** Full LSP server (Go) for multi-editor support.

---

## 🧪 Versioned Milestones

| Milestone | Goal | ETA |
|------------|------|------|
| **v0.1.0** | Lint core + adapters | 3 weeks |
| **v0.2.0** | Probe engine (CDP metrics) | +3 weeks |
| **v0.3.0** | CI integration (reports + baselines) | +2 weeks |
| **v0.4.0** | HUD + overlay | +2 weeks |
| **v0.5.0** | Editor (VS Code) + autofix | +3 weeks |
| **v1.0.0** | Tracelet Cloud + historical dashboard | +4 weeks |

---

## ✅ Version Checklists

### v0.1.0 — Lint Core ✅

- [x] CLI + config loader
- [x] `route-initial-js`, `font-display`, `unoptimized-image` rules (+ `missing-alt-text`, `missing-image-dimensions`, `missing-preconnect`, `render-blocking-resources`)
- [x] Next/Vite adapters
- [x] Table + JSON output
- [x] Exit codes for CI

### v0.2.0 — Probe Core ✅

- [x] CDP runner (chromedp)
- [x] Collect FCP, LCP, CLS, TBT-Lite, FSI
- [x] Deterministic profiles
- [x] JSON output for CI + HUD

### v0.3.0 — CI ✅

- [x] GitHub Action template
- [x] Markdown comment generator
- [x] Baseline diffing

### v0.4.0 — HUD ✅

- [x] WebSocket overlay
- [x] Live reload + per-route metrics (live Web Vitals + React re-render counts)
- [x] HUD toggle in config (`hud.enabled` / `hud.port`)

### v0.5.0 — Editor 🟡 (Phase 1 shipped)

- [x] VS Code shell → `tracelet lint --json`
- [x] Quick Fix actions
- [x] Status bar + probe integration
- [ ] Move to tracelet-lsp (Phase 2) — not started

### Deferred / not yet built

- [ ] Astro adapter (only Next.js + Vite ship today)
- [ ] `tracelet-lsp` — Phase 2 multi-editor LSP server
- [ ] All of v1.0.0 (Cloud) below

### v1.0.0 — Cloud

- [ ] Baseline sync API
- [ ] Dashboard UI
- [ ] Auth + org/team management
- [ ] Webhook notifications

---

## ⚙️ Tech Stack

| Layer | Tech |
|-------|------|
| Core CLI | Go 1.23+ |
| CDP Client | `chromedp` |
| HUD | Go WS + Vanilla JS |
| Adapters | JS (Next/Vite/Astro) |
| Config | JSON |
| LSP | Go (Phase 2) |
| Editor UI | VS Code API (TypeScript) |
| Packaging | GoReleaser |
| Optional Dashboard | Next.js + Tailwind |

---

## 📘 Summary

Tracelet isn’t another framework or metric viewer —
it’s the **performance layer of your development loop**.

It’s built for speed, clarity, and integration:

- **Lint** performance budgets like code.
- **Probe** runtime behavior in seconds.
- **See** results live in your editor or CI.

> **Tracelet** — performance you can lint.
