# Configuration Reference

Tracelet uses a centralized configuration file: `tracelet.config.json`.

## Quick Start

Generate a default config:

```bash
# With npm
npx tracelet init

# Or if built from source
./tracelet init
```

This creates:

```json
{
  "extends": ["recommended"],
  "budgets": {
    "default": { "initialJs": "35kb", "thirdPartyJs": "50kb" },
    "/": { "initialJs": "10kb" }
  },
  "rules": {
    "route-initial-js": "error",
    "unoptimized-image": "warn",
    "font-display": "info"
  },
  "probe": { "profile": "desktop", "runs": 1 },
  "collect": { "framework": "next", "statsFile": ".tracelet/stats.json" }
}
```

## Schema

### Top-Level Properties

| Property | Type | Description |
|----------|------|-------------|
| `extends` | `string[]` | Config presets to extend (e.g., `["recommended"]`) |
| `budgets` | `object` | Per-route performance budgets |
| `rules` | `object` | Rule severity overrides |
| `probe` | `object` | Probe configuration |
| `collect` | `object` | Framework adapter configuration |

### Budgets

Budgets define size limits for JavaScript bundles per route.

```json
{
  "budgets": {
    "default": {
      "initialJs": "35kb",
      "thirdPartyJs": "50kb"
    },
    "/": {
      "initialJs": "10kb"
    },
    "/product": {
      "initialJs": "25kb",
      "thirdPartyJs": "40kb"
    }
  }
}
```

**Budget Properties:**
- `initialJs`: Maximum gzipped size for initial JavaScript (route entry + shared chunks)
- `thirdPartyJs`: Maximum gzipped size for third-party libraries

**Size Formats:**
- `"35kb"` or `"35KB"` - kilobytes (1 KB = 1024 bytes)
- `"50kib"` - kibibytes
- `"1024b"` - bytes

**Route Matching:**
- Exact match: `"/"`, `"/about"`, `"/product/123"`
- Dynamic routes: Use bracket notation like `"/[locale]"` or `"/[slug]"`

### Rules

Configure rule severity levels:

```json
{
  "rules": {
    "route-initial-js": "error",
    "unoptimized-image": "warn",
    "font-display": "info"
  }
}
```

**Severity Levels:**
- `"off"` - Disable the rule
- `"info"` - Information only (shown in Problems panel)
- `"warn"` - Warning (shown as yellow squiggle)
- `"error"` - Error (shown as red squiggle, fails CI)

See [Rules Reference](./rules.md) for all available rules.

### Probe

Configure the `tracelet probe` command:

```json
{
  "probe": {
    "profile": "desktop",
    "runs": 1
  }
}
```

**Properties:**
- `profile`: `"desktop"` or `"mobile"` - Network/CPU throttling profile
- `runs`: `number` - Number of runs to average (default: 1)

### Collect

Configure framework adapter for stats collection:

```json
{
  "collect": {
    "framework": "next",
    "statsFile": ".tracelet/stats.json"
  }
}
```

**Properties:**
- `framework`: `"next"` or `"vite"` - Framework to use for collecting build stats
- `statsFile`: `string` - Path to stats output file (default: `.tracelet/stats.json`)

## Extends

Use presets to share common configurations:

```json
{
  "extends": ["recommended"]
}
```

Currently supported presets:
- `"recommended"` - Default sensible rules and budgets

## Example Configurations

### Strict Mode

```json
{
  "extends": ["recommended"],
  "budgets": {
    "default": { "initialJs": "20kb", "thirdPartyJs": "30kb" }
  },
  "rules": {
    "route-initial-js": "error",
    "unoptimized-image": "error",
    "font-display": "warn"
  }
}
```

### Per-Route Budgets

```json
{
  "budgets": {
    "default": { "initialJs": "35kb" },
    "/": { "initialJs": "10kb" },
    "/dashboard": { "initialJs": "50kb" },
    "/product/[id]": { "initialJs": "30kb" }
  }
}
```

### CI-Optimized

```json
{
  "probe": {
    "profile": "mobile",
    "runs": 3
  },
  "rules": {
    "route-initial-js": "error"
  }
}
```

## CLI Override

You can override the config path:

```bash
npx tracelet lint --config custom/path/config.json
```

## Validation

Tracelet validates your config on load:
- Required fields must be present
- Framework must be `"next"` or `"vite"`
- Budget sizes must be parseable
- Rule severities must be valid

Errors are shown immediately with helpful messages.

