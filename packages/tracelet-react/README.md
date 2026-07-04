# tracelet-react

Tracks React component **re-renders** and surfaces them in the Tracelet HUD's Components tab.

It works by installing the React DevTools global hook (`__REACT_DEVTOOLS_GLOBAL_HOOK__`) and
counting each commit via `onCommitFiberRoot` — the same mechanism React DevTools uses. Components
that bail out (e.g. wrapped in `React.memo` with unchanged props) are correctly **not** counted.

## Installation

```bash
npm install -D tracelet-react
```

## Usage — must load before React

> **Important:** the hook must be installed **before React initializes**, because React reads
> `__REACT_DEVTOOLS_GLOBAL_HOOK__` only once, at startup. Loading it afterwards tracks nothing.

### Bundled app (Next.js, Vite, CRA, …)

Import it as the **first line** of your dev entry point:

```javascript
// main.jsx / index.tsx — top of file, before any React import
if (import.meta.env.DEV) {
  await import('tracelet-react');
}
```

or with CommonJS:

```javascript
if (process.env.NODE_ENV === 'development') {
  require('tracelet-react');
}
```

### Plain HTML / when running the HUD

Add the hook served by the HUD to `<head>`, **before** your app bundle:

```html
<head>
  <script src="http://localhost:3111/hook.js"></script>
</head>
```

The HUD's `overlay.js` also injects the instrumentation as a best-effort fallback, but that runs
after your app loads — it only works if the React DevTools extension is already present. For
reliable tracking, use one of the two methods above.

## API

Exposes `window.__traceletReactInstrumentation`:

```javascript
window.__traceletReactInstrumentation.getComponents();   // [{ name, renderCount }, ...]
window.__traceletReactInstrumentation.getRenderCounts();  // { ComponentName: count, ... }
window.__traceletReactInstrumentation.reset();            // clear counts
```

Dev only — do not ship to production.
