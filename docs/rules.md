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

### `missing-image-dimensions`

Flags images missing `width` and `height` attributes.

**Default Severity:** `warn`

**What it checks:**
- `<img>` tags without both `width` and `height` attributes

**Example violation:**
```html
<img src="hero.jpg" alt="Hero">
<!-- Should have: <img src="hero.jpg" alt="Hero" width="800" height="600"> -->
```

**Why it matters:**
Without dimensions, browsers can't reserve space for images, causing layout shift (CLS) when images load.

**Note:** Cannot be autofixed as it requires actual image dimensions. Use image processing tools or manually add dimensions.

### `render-blocking-resources`

Flags CSS and JavaScript resources that block page rendering.

**Default Severity:** `warn`

**What it checks:**
- `<script>` tags in `<head>` without `defer`, `async`, or `type="module"`
- `<link rel="stylesheet">` tags in `<head>` without `rel="preload"` or `media="print"`

**Example violations:**
```html
<head>
  <script src="analytics.js"></script> <!-- Should have defer or async -->
  <link rel="stylesheet" href="styles.css"> <!-- Should be preloaded -->
</head>
```

**Quick Fix:**
Autofix can add `defer` to scripts. For CSS, manually add `rel="preload"` for critical stylesheets.

**Best Practices:**
- Use `defer` for scripts that don't need to run immediately
- Use `async` for independent scripts (analytics, widgets)
- Preload critical CSS with `<link rel="preload" as="style" href="critical.css">`

### `missing-preconnect`

Flags external domains used without preconnect or dns-prefetch.

**Default Severity:** `info`

**What it checks:**
- External domains referenced in `href`, `src`, or `action` attributes
- Checks if domain has `<link rel="preconnect">` or `<link rel="dns-prefetch">` in `<head>`

**Example violation:**
```html
<head>
  <!-- Missing preconnect for external domain -->
</head>
<body>
  <img src="https://cdn.example.com/image.jpg">
</body>
```

**Recommended fix:**
```html
<head>
  <link rel="preconnect" href="https://cdn.example.com">
</head>
```

**Best Practices:**
- Use `preconnect` for critical third-party domains (fonts, APIs, CDNs)
- Use `dns-prefetch` for less critical domains
- Add preconnect links early in `<head>` for better performance

### `missing-alt-text`

Flags images missing `alt` attribute.

**Default Severity:** `warn`

**What it checks:**
- `<img>` tags without `alt` attribute

**Example violation:**
```html
<img src="logo.png">
<!-- Should have: <img src="logo.png" alt="Company Logo"> -->
```

**Quick Fix:**
Autofix adds `alt=""` (empty alt for decorative images). Update manually with descriptive text for meaningful images.

**Best Practices:**
- Always include `alt` text for accessibility
- Use empty `alt=""` for purely decorative images
- Use descriptive text for images that convey information

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
    "font-display": "warn",
    "missing-image-dimensions": "error",
    "render-blocking-resources": "error",
    "missing-preconnect": "warn",
    "missing-alt-text": "warn",
    "react-inline-props": "warn",
    "react-missing-memo": "info",
    "react-unstable-props": "warn"
  }
}
```

For development/learning:

```json
{
  "rules": {
    "route-initial-js": "warn",
    "unoptimized-image": "info",
    "font-display": "info",
    "missing-image-dimensions": "info",
    "render-blocking-resources": "info",
    "missing-preconnect": "off",
    "missing-alt-text": "info",
    "react-inline-props": "info",
    "react-missing-memo": "off",
    "react-unstable-props": "info"
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

### `react-inline-props`

Flags inline function/object literals in JSX props that can cause unnecessary rerenders.

**Default Severity:** `warn`

**What it checks:**
- Inline arrow functions in props: `<Component onClick={() => {}} />`
- Inline object literals in props: `<Component style={{ color: 'red' }} />`

**Example violations:**
```tsx
// Inline arrow function
<Button onClick={() => handleClick(id)} />

// Inline object literal
<Component data={{ key: value }} />
```

**Recommended fixes:**
```tsx
// Use useCallback for functions
const handleClick = useCallback(() => {
  handleClickInternal(id);
}, [id]);

// Move objects outside render or use useMemo
const data = useMemo(() => ({ key: value }), [value]);
```

**Best Practices:**
- Use `useCallback` for event handlers passed as props
- Use `useMemo` for objects/arrays passed as props
- Move static objects outside the component render

### `react-missing-memo`

Flags components that receive props but aren't memoized with `React.memo`.

**Default Severity:** `info`

**What it checks:**
- Components that receive props but aren't wrapped with `React.memo`
- Only flags components that actually receive props

**Example violation:**
```tsx
// Component receives props but isn't memoized
function ProductCard({ title, price }) {
  return <div>{title} - ${price}</div>;
}
```

**Recommended fix:**
```tsx
// Wrap with React.memo to prevent unnecessary rerenders
const ProductCard = React.memo(function ProductCard({ title, price }) {
  return <div>{title} - ${price}</div>;
});
```

**Best Practices:**
- Use `React.memo` for components that render frequently
- Only memoize if props change infrequently
- Don't memoize components that receive new props on every render

### `react-unstable-props`

Flags unstable props (inline functions/objects) passed from parent components that cause child rerenders.

**Default Severity:** `warn`

**What it checks:**
- Parent components passing inline functions to children
- Parent components passing inline objects to children

**Example violation:**
```tsx
function Parent() {
  return (
    <Child
      onClick={() => handleClick()}  // Inline function
      data={{ key: 'value' }}        // Inline object
    />
  );
}
```

**Recommended fix:**
```tsx
function Parent() {
  const handleClick = useCallback(() => {
    handleClickInternal();
  }, []);

  const data = useMemo(() => ({ key: 'value' }), []);

  return <Child onClick={handleClick} data={data} />;
}
```

**Best Practices:**
- Move event handlers to `useCallback` in parent components
- Move objects/arrays to `useMemo` in parent components
- This prevents child components from rerendering when parent rerenders

## Future Rules

Planned rules for future releases:
- `large-lib` - Flag oversized third-party libraries
- `unused-css` - Detect unused CSS rules
- `duplicate-chunks` - Find duplicate code across chunks

## Contributing Rules

See [Contributing Guide](../CONTRIBUTING.md) for how to add custom rules.

