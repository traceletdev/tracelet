# @traceletdev/vite

Tracelet plugin for Vite projects.

## Installation

```bash
npm install -D @traceletdev/cli @traceletdev/vite
```

## Usage

Add the plugin to your `vite.config.js`:

```js
import { defineConfig } from 'vite';
import tracelet from '@traceletdev/vite';

export default defineConfig({
  plugins: [
    // your other plugins
    tracelet(),
  ],
});
```

The plugin automatically runs during builds and generates `.tracelet/stats.json` which is used by `tracelet lint` to check performance budgets.

## How it works

The plugin hooks into Vite's build process and:

- Analyzes entry chunks during bundle generation
- Calculates gzip-compressed JavaScript sizes
- Identifies third-party JavaScript bundles
- Generates statistics for tracelet linting

## Configuration

Ensure you have a `tracelet.config.json` file in your project root. See the [main tracelet documentation](../../README.md) for configuration options.

