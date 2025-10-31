# Rules Reference

Tracelet includes several built-in performance rules to catch common issues.

## Rule Catalog

### `route-initial-js`

Enforces per-route JavaScript bundle size budgets.

**Default Severity:** `error`

**What it checks:**
- Total gzipped JavaScript size for each route's initial load
- Compares against budget defined in `tracelet.config.json`
- Accounts for route-specific budgets if defined

**Example:**
```json
{
  "budgets": {
    "/": { "initialJs": "10kb" }
  }
}
```

If `/` route has 15KB of JS, this rule will trigger an error.

**Output:**
```
❌ [route-initial-js] / — 15KB JS (over by 5KB)
```

### `unoptimized-image`

Flags images missing optimization attributes.

**Default Severity:** `warn`

**What it checks:**
- `<img>` tags without `width` and `height` attributes
- `<img>` tags without `loading="lazy"` attribute

**Example violations:**
```html
<!-- Missing width/height -->
<img src="hero.jpg" alt="Hero">

<!-- Missing lazy loading -->
<img src="thumbnail.jpg" alt="Thumb" width="100" height="100">
```

**Quick Fix:**
VS Code extension provides Quick Fix to add `loading="lazy"` automatically.

**Best Practices:**
- Always specify `width` and `height` to prevent layout shift
- Use `loading="lazy"` for below-the-fold images

### `font-display`

Flags `@font-face` declarations missing `font-display`.

**Default Severity:** `info`

**What it checks:**
- CSS `@font-face` rules without `font-display` property

**Example violation:**
```css
@font-face {
  font-family: 'Inter';
  src: url('inter.woff2') format('woff2');
  /* Missing font-display */
}
```

**Recommended fix:**
```css
@font-face {
  font-family: 'Inter';
  src: url('inter.woff2') format('woff2');
  font-display: swap;
}
```

**Quick Fix:**
VS Code extension provides Quick Fix to add `font-display: swap` automatically.

**font-display values:**
- `swap` - Show fallback immediately, swap when font loads (recommended)
- `optional` - Use fallback if font not cached
- `block` - Hide text until font loads (not recommended)

## Configuring Rules

### Disable a Rule

```json
{
  "rules": {
    "font-display": "off"
  }
}
```

### Change Severity

```json
{
  "rules": {
    "unoptimized-image": "error",
    "font-display": "warn"
  }
}
```

### Recommended Settings

For strict performance requirements:

```json
{
  "rules": {
    "route-initial-js": "error",
    "unoptimized-image": "error",
    "font-display": "warn"
  }
}
```

For development/learning:

```json
{
  "rules": {
    "route-initial-js": "warn",
    "unoptimized-image": "info",
    "font-display": "info"
  }
}
```

## Rule Output Formats

### CLI Table

```
Route        JS(gzip)  Verdict
/            9KB       ✅
/product     47KB      ❌ over budget

Diagnostics:
❌ [route-initial-js] /product — 47KB JS (over by 12KB)
⚠️  [unoptimized-image] <img> missing width/height in src/components/Hero.tsx
```

### JSON

```json
{
  "results": [
    {
      "ruleId": "route-initial-js",
      "level": "error",
      "route": "/product",
      "detail": "47KB JS (over by 12KB)"
    },
    {
      "ruleId": "unoptimized-image",
      "level": "warn",
      "detail": "<img> missing width/height in src/components/Hero.tsx"
    }
  ]
}
```

## Exit Codes

Rules affect CLI exit codes:

- **Exit 0**: All rules passed or only `info`/`warn` (unless `--ci` flag)
- **Exit 1**: Warnings present (only with `--ci` flag)
- **Exit 2**: Errors present

```bash
# CI mode: exit 1 on warnings
tracelet lint --ci

# Normal mode: exit 0 even with warnings
tracelet lint
```

## Future Rules

Planned rules for future releases:
- `large-lib` - Flag oversized third-party libraries
- `unused-css` - Detect unused CSS rules
- `duplicate-chunks` - Find duplicate code across chunks

## Contributing Rules

See [Contributing Guide](../CONTRIBUTING.md) for how to add custom rules.

