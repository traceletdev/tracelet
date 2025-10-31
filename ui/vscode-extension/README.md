# Tracelet VS Code Extension

Performance linting and diagnostics for your routes directly in VS Code.

## Features

- **Lint on Save**: Automatically lints your routes when you save files
- **Problems Panel**: Shows performance diagnostics inline
- **Quick Fixes**: Automatically fix common issues:
  - Add `loading="lazy"` to images
  - Add `font-display: swap` to font-face rules
- **Status Bar**: Shows total JS bundle size at a glance
- **Probe Command**: Run performance probes from the command palette

## Setup

1. Install dependencies:
```bash
cd ui/vscode-extension
npm install
```

2. Compile:
```bash
npm run compile
```

3. Press F5 in VS Code to launch extension development host

## Configuration

- `tracelet.binaryPath`: Path to tracelet binary (leave empty to auto-detect)
- `tracelet.debounceMs`: Debounce delay for lint on save (default: 600ms)
- `tracelet.enableOnType`: Enable linting on type (may be slow, default: false)
- `tracelet.probe.profile`: Profile for probe command (desktop/mobile, default: desktop)

## Usage

- Linting happens automatically on save
- Use Command Palette (`Cmd+Shift+P` / `Ctrl+Shift+P`):
  - `Tracelet: Lint Changed` - Manually lint current file
  - `Tracelet: Probe Current Route` - Probe a URL for performance metrics
  - `Tracelet: Open Config` - Open tracelet.config.json

## Quick Fixes

Hover over a diagnostic and click the lightbulb to see available quick fixes.
