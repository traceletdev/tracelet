'use strict';

/**
 * Tracelet React Instrumentation
 *
 * Counts component re-renders by installing the React DevTools global hook
 * (window.__REACT_DEVTOOLS_GLOBAL_HOOK__) and inspecting each commit via
 * onCommitFiberRoot. This is the only reliable way to observe renders in a
 * real (bundled / JSX-runtime) app — patching React.createElement does not
 * work because compiled JSX never calls the global React.createElement.
 *
 * HARD REQUIREMENT: this must run BEFORE React initializes, because React
 * only reads __REACT_DEVTOOLS_GLOBAL_HOOK__ once, at module init. Load it via
 * <script src=".../hook.js"> in <head> before your bundle, or as the first
 * import in your app entry (dev only).
 *
 * ponytail: canonical source. Kept byte-identical with
 * packages/tracelet-react/index.js — edit both together.
 */

(function () {
  if (typeof window === 'undefined') return;
  if (window.__traceletReactInstrumentation) return;

  var DEBUG = false; // set window.__TRACELET_DEBUG = true to trace
  function log() {
    if (window.__TRACELET_DEBUG || DEBUG) {
      try { console.log.apply(console, ['[tracelet]'].concat([].slice.call(arguments))); } catch (e) {}
    }
  }

  var PerformedWork = 0b1; // React fiber flag set when beginWork ran this commit
  var renderCounts = {};   // display name -> cumulative render count

  function getDisplayName(fiber) {
    var type = fiber.type;
    if (typeof type === 'function') return type.displayName || type.name || null;
    if (typeof type === 'string') return null; // host component (div/span/...)
    if (type && typeof type === 'object') {
      // React.memo (type.type) or forwardRef (type.render)
      if (type.displayName) return type.displayName;
      var inner = type.type || type.render;
      if (typeof inner === 'function') return inner.displayName || inner.name || null;
    }
    return null;
  }

  // Walk the committed fiber tree and count every composite component that
  // performed work this commit (freshly mounted, or PerformedWork flag set).
  function onCommit(root) {
    if (!root || !root.current) return;
    var stack = [root.current];
    var guard = 0;
    while (stack.length) {
      var fiber = stack.pop();
      if (!fiber) continue;
      var name = getDisplayName(fiber);
      if (name) {
        var flags = (fiber.flags !== undefined ? fiber.flags : fiber.effectTag) || 0;
        var rendered = fiber.alternate == null || (flags & PerformedWork) !== 0;
        if (rendered) renderCounts[name] = (renderCounts[name] || 0) + 1;
      }
      if (fiber.child) stack.push(fiber.child);
      if (fiber.sibling) stack.push(fiber.sibling);
      if (++guard > 200000) break; // safety against cycles
    }
  }

  function safeCommit(root) {
    try { onCommit(root); } catch (e) { log('commit error', e); }
  }

  // Install (or wrap) the DevTools global hook.
  var existing = window.__REACT_DEVTOOLS_GLOBAL_HOOK__;
  if (existing) {
    // Real DevTools (or another tool) already present — chain onto it so we
    // don't break it. We only see commits from this point forward.
    var origOnCommit = existing.onCommitFiberRoot;
    existing.onCommitFiberRoot = function (id, root) {
      safeCommit(root);
      if (typeof origOnCommit === 'function') return origOnCommit.apply(this, arguments);
    };
    log('wrapped existing DevTools hook');
  } else {
    var renderers = new Map();
    var uid = 0;
    window.__REACT_DEVTOOLS_GLOBAL_HOOK__ = {
      renderers: renderers,
      supportsFiber: true,
      inject: function (renderer) { var id = ++uid; renderers.set(id, renderer); return id; },
      onCommitFiberRoot: function (id, root) { safeCommit(root); },
      onCommitFiberUnmount: function () {},
      on: function () {},
      emit: function () {},
      checkDCE: function () {}
    };
    log('installed minimal DevTools hook');
  }

  window.__traceletReactInstrumentation = {
    getComponents: function () {
      var out = [];
      for (var name in renderCounts) {
        out.push({ name: name, renderCount: renderCounts[name] });
      }
      return out;
    },
    getRenderCounts: function () { return renderCounts; },
    reset: function () { renderCounts = {}; }
  };
})();

// Export for module systems (harmless/no-op in the browser <script> case).
if (typeof module !== 'undefined' && module.exports) {
  module.exports = (typeof window !== 'undefined') ? window.__traceletReactInstrumentation : {};
}
