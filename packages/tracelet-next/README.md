# @traceletdev/next

Tracelet plugin for Next.js projects.

## Installation

```bash
npm install -D @traceletdev/cli @traceletdev/next
```

## Usage

After installing, add a postbuild script to your `package.json`:

```json
{
  "scripts": {
    "postbuild": "tracelet-next collect"
  }
}
```

Or run manually after building:

```bash
npm run build
npx tracelet-next collect
```

This will generate `.tracelet/stats.json` which is used by `tracelet lint` to check performance budgets.

## How it works

The plugin reads Next.js build manifests (`.next/build-manifest.json`, `.next/app-build-manifest.json`, etc.) to:

- Detect all routes (including dynamic routes)
- Calculate gzip-compressed JavaScript sizes per route
- Identify third-party JavaScript bundles
- Generate statistics for tracelet linting

## Configuration

Ensure you have a `tracelet.config.json` file in your project root. See the [main tracelet documentation](../../README.md) for configuration options.

