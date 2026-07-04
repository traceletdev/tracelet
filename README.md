# Tracelet

> **Small traces. Big clarity.**

Tracelet is a modern performance toolkit that brings **Lighthouse-level insight** and **ESLint-level ergonomics** into one seamless workflow.

[![Go Version](https://img.shields.io/badge/go-1.23+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## Features

- **🚀 Fast** - Every operation completes in seconds
- **🎯 Deterministic** - Reproducible metrics regardless of environment
- **🔧 Integrated** - Fits into your dev loop, editor, and CI
- **👁️ Transparent** - See what's happening, not just summaries

## Quick Start

### Install via npm (Recommended)

```bash
# Install tracelet CLI
npm install -D tracelet

# Install framework plugin (choose one)
npm install -D tracelet-next    # for Next.js
npm install -D tracelet-vite      # for Vite

# Initialize config
npx tracelet init

# Lint
npx tracelet lint

# Probe
npx tracelet probe http://localhost:3000
```

See [Framework Integration](#framework-integration) below for setup details.

### Build from source

```bash
# Build
go build -o tracelet ./cmd/tracelet

# Initialize
./tracelet init

# Lint
./tracelet lint

# Probe
./tracelet probe http://localhost:3000
```

## What's Included

### 📋 Lint
Performance budgets as code. Enforce route-level JavaScript limits with ESLint-style rules.

```bash
tracelet lint
# Route        JS(gzip)  Verdict
# /            9KB       ✅
# /product     47KB      ❌ over budget
```

### 🔍 Probe
Chrome-based performance audits in 2-3 seconds. Collect TTFB, FCP, LCP, CLS, TBT-Lite, FSI.

```bash
tracelet probe http://localhost:3000 --profile mobile
```

### 🖥️ HUD
Real-time overlay for development. See live performance feedback in your browser.

```bash
tracelet hud
# Then add <script src="http://localhost:3111/overlay.js"></script> to your app
```

The overlay shows a **Routes** tab (lint budgets), a **Metrics** tab (live Web Vitals), and a
**Components** tab (React re-render counts). React tracking needs its hook loaded *before* React —
add this to `<head>`, before your app bundle:

```html
<script src="http://localhost:3111/hook.js"></script>
```

See [`tracelet-react`](./packages/tracelet-react/README.md) for the bundled-app (`import`) setup.

### 🔄 CI
Automated checks with baseline comparison. GitHub Action ready.

```bash
tracelet ci --compare .tracelet/baseline.json --format markdown
```

### 📝 VS Code
In-editor diagnostics with quick fixes. See issues as you code.

Install the extension and get instant feedback on save.

## Documentation

- [Overview](./docs/overview.md) - Philosophy and features
- [Configuration](./docs/config.md) - Config reference
- [Rules](./docs/rules.md) - Rule catalog
- [API Reference](./docs/api.md) - CLI and programmatic APIs
- [Framework Adapters](./docs/adapters.md) - Next.js and Vite integration

## Framework Integration

### Next.js

1. Install packages:
   ```bash
   npm install -D tracelet tracelet-next
   ```

2. Add postbuild script to `package.json`:
   ```json
   {
     "scripts": {
       "postbuild": "node node_modules/tracelet-next/collect.js"
     }
   }
   ```

3. Build your app:
   ```bash
   npm run build
   # Stats automatically collected
   ```

4. Lint with tracelet:
   ```bash
   npx tracelet lint
   ```

### Vite

1. Install packages:
   ```bash
   npm install -D tracelet tracelet-vite
   ```

2. Add plugin to `vite.config.js`:
   ```js
   import { defineConfig } from 'vite';
   import tracelet from 'tracelet-vite';

   export default defineConfig({
     plugins: [
       tracelet(),
       // your other plugins
     ],
   });
   ```

3. Build your app:
   ```bash
   npm run build
   # Stats automatically collected during build
   ```

4. Lint with tracelet:
   ```bash
   npx tracelet lint
   ```

## Adapters (Legacy)

For projects not using npm packages, see adapter files in `adapters/` directory:
- **Next.js** - `node adapters/next-collect.js` after build
- **Vite** - Use `adapters/vite-plugin-tracelet.cjs` in `vite.config.js`

## Examples

Check `tests/fixtures/` for example Next.js and Vite projects.

## Development

```bash
# Run tests
go test ./tests/integration -v

# Build CLI
go build -o tracelet ./cmd/tracelet

# Build VS Code extension
cd ui/vscode-extension
npm install
npm run compile
```

## Project Structure

```
tracelet/
├─ cmd/tracelet/        # CLI entrypoint
├─ internal/            # Core packages
│  ├─ config/          # Config loader
│  ├─ lint/            # Linting engine
│  ├─ probe/           # Chrome probe
│  ├─ hud/             # HUD server
│  ├─ ci/              # CI helpers
│  └─ reporters/       # Output formatters
├─ adapters/            # Framework adapters
├─ ui/vscode-extension/ # VS Code extension
├─ tests/              # Tests and fixtures
└─ docs/               # Documentation
```

## License

MIT

## Contributing

Contributions welcome! See [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines and
our [Code of Conduct](./CODE_OF_CONDUCT.md). Release history is in
[CHANGELOG.md](./CHANGELOG.md).

---

**Tracelet** — performance you can lint.

