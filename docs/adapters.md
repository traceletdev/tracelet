# Framework Adapters

Tracelet integrates with modern frameworks via npm packages that automatically collect route statistics during builds.

## Installation

### Main Package

```bash
npm install -D @traceletdev/cli
```

### Framework Plugins

```bash
# For Next.js
npm install -D @traceletdev/next

# For Vite
npm install -D @traceletdev/vite
```

## Next.js Integration

### Setup

1. Install packages:

   ```bash
   npm install -D @traceletdev/cli @traceletdev/next
   ```

2. Add postbuild script to `package.json`:

   ```json
   {
     "scripts": {
       "postbuild": "tracelet-next collect"
     }
   }
   ```

3. Build your app:

   ```bash
   npm run build
   ```

   The adapter automatically runs after the build completes and generates `.tracelet/stats.json`.

### How It Works

The `@traceletdev/next` package provides a script that:

- Reads Next.js build manifests (`.next/build-manifest.json`, `.next/app-build-manifest.json`)
- Detects all routes including:
  - Static routes (`/`, `/about`)
  - Dynamic routes (`/[locale]`, `/[locale]/money2020`)
  - App Router and Pages Router routes
- Calculates gzip-compressed JavaScript sizes per route
- Identifies third-party JavaScript bundles
- Generates `.tracelet/stats.json` for linting

### Programmatic Usage

You can also call the adapter programmatically:

```js
const collect = require('@traceletdev/next');
collect(); // Collects stats and writes to .tracelet/stats.json
```

## Vite Integration

### Setup

1. Install packages:

   ```bash
   npm install -D @traceletdev/cli @traceletdev/vite
   ```

2. Add plugin to `vite.config.js`:

   ```js
   import { defineConfig } from 'vite';
   import tracelet from '@traceletdev/vite';

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
   ```

   The plugin automatically runs during the build and generates `.tracelet/stats.json`.

### How It Works

The `@traceletdev/vite` plugin:

- Hooks into Vite's `generateBundle` phase
- Analyzes entry chunks and calculates gzip-compressed sizes
- Identifies third-party JavaScript bundles (vendor chunks)
- Generates `.tracelet/stats.json` for linting

### Plugin Options

Currently, the plugin uses default settings. Future versions may support:

```js
tracelet({
  // Custom output path
  outputFile: '.tracelet/stats.json',

  // Include/exclude patterns
  include: ['**/*.js'],
  exclude: ['node_modules/**'],
});
```

## Legacy Adapter Scripts

For projects not using npm packages, you can use the adapter scripts directly:

### Next.js

```bash
# After building your Next.js app
node node_modules/@traceletdev/cli/adapters/next-collect.js
```

### Vite

Copy `adapters/vite-plugin-tracelet.cjs` to your project and use it in `vite.config.js`:

```js
const tracelet = require('./adapters/vite-plugin-tracelet.cjs');
```

## Verification

After running builds, verify that stats were collected:

```bash
# Check stats file exists
cat .tracelet/stats.json

# Lint with tracelet
npx tracelet lint
```

The stats file should contain route information:

```json
{
  "routes": [
    {
      "path": "/",
      "jsGzipBytes": 45000,
      "thirdPartyJsBytes": 12000
    },
    {
      "path": "/[locale]",
      "jsGzipBytes": 196000,
      "thirdPartyJsBytes": 50000
    }
  ]
}
```

## Troubleshooting

### Stats file not generated

- **Next.js**: Ensure `postbuild` script runs after successful build
- **Vite**: Check that plugin is in `plugins` array (not just imported)
- **Both**: Verify `.next` or `dist` directory exists after build

### Zero byte sizes

- Check that build completed successfully
- Verify manifest files exist (`.next/build-manifest.json` for Next.js)
- Ensure JavaScript chunks are being generated

### Missing routes

- For Next.js: Check that routes are in `app-build-manifest.json` or `build-manifest.json`
- Dynamic routes may not appear if they're not prerendered
- API routes are excluded by design

## Advanced Usage

### Custom Collection Timing

You can trigger collection manually:

```bash
# Next.js
npx tracelet-next collect

# Or programmatically
node -e "require('@traceletdev/next')()"
```

### CI/CD Integration

In CI pipelines, ensure adapters run after builds:

```yaml
# GitHub Actions example
- name: Build Next.js app
  run: npm run build

- name: Collect Tracelet stats
  run: npx tracelet-next collect

- name: Run Tracelet lint
  run: npx tracelet lint
```

## Contributing

To add support for a new framework:

1. Create adapter in `packages/tracelet-<framework>/` (published as `@traceletdev/<framework>`)
2. Follow the stats schema: `{ routes: [{ path, jsGzipBytes, thirdPartyJsBytes }] }`
3. Write to `.tracelet/stats.json`
4. Document setup in this file
